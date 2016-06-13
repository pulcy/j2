// Copyright (c) 2016 Pulcy.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fleet

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	etcd "github.com/coreos/etcd/client"
	"github.com/coreos/fleet/client"
	"github.com/coreos/fleet/job"
	"github.com/coreos/fleet/machine"
	"github.com/coreos/fleet/pkg"
	"github.com/coreos/fleet/registry"
	"github.com/coreos/fleet/schema"
	"github.com/coreos/fleet/ssh"
	aerr "github.com/ewoutp/go-aggregate-error"
	"github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("fleet")
)

type FleetTunnel struct {
	FleetConfig
	cAPI          client.API
	machineStates map[string]*machine.MachineState
}

func NewTunnel(config FleetConfig) (*FleetTunnel, error) {
	if config.Tunnel != "" && !strings.Contains(config.Tunnel, ":") {
		config.Tunnel = net.JoinHostPort(config.Tunnel, "22")
	}
	cAPI, err := getHTTPClient(config)
	if err != nil {
		return nil, maskAny(err)
	}
	return &FleetTunnel{
		FleetConfig: config,
		cAPI:        cAPI,
	}, nil
}

func (f *FleetTunnel) findUnits(names []string) (sus []schema.Unit, err error) {
	units, err := f.cAPI.Units()
	if err != nil {
		return nil, err
	}

	uMap := make(map[string]*schema.Unit, len(units))
	for _, u := range units {
		u := u
		uMap[u.Name] = u
	}

	var filtered []schema.Unit
	for _, v := range names {
		u, ok := uMap[v]
		if !ok {
			continue
		}
		filtered = append(filtered, *u)
	}

	return filtered, nil
}

// tryWaitForUnitStates tries to wait for units to reach the desired state.
// It takes 5 arguments, the units to wait for, the desired state, the
// desired JobState, how many attempts before timing out and a writer
// interface.
// tryWaitForUnitStates polls each of the indicated units until they
// reach the desired state. If maxAttempts is negative, then it will not
// wait, it will assume that all units reached their desired state.
// If maxAttempts is zero tryWaitForUnitStates will retry forever, and
// if it is greater than zero, it will retry up to the indicated value.
// It returns 0 on success or 1 on errors.
func (f *FleetTunnel) tryWaitForUnitStates(units []string, state string, js job.JobState, maxAttempts int, out io.Writer) error {
	// We do not wait just assume we reached the desired state
	if maxAttempts <= -1 {
		for _, name := range units {
			out.Write([]byte(fmt.Sprintf("Triggered unit %s %s\n", name, state)))
		}
		return nil
	}

	errchan := f.waitForUnitStates(units, js, maxAttempts, out)
	var ae aerr.AggregateError
	for err := range errchan {
		ae.Add(maskAny(err))
	}

	if !ae.IsEmpty() {
		return maskAny(&ae)
	}

	return nil
}

// waitForUnitStates polls each of the indicated units until each of their
// states is equal to that which the caller indicates, or until the
// polling operation times out. waitForUnitStates will retry forever, or
// up to maxAttempts times before timing out if maxAttempts is greater
// than zero. Returned is an error channel used to communicate when
// timeouts occur. The returned error channel will be closed after all
// polling operation is complete.
func (f *FleetTunnel) waitForUnitStates(units []string, js job.JobState, maxAttempts int, out io.Writer) chan error {
	errchan := make(chan error)
	var wg sync.WaitGroup
	for _, name := range units {
		wg.Add(1)
		go f.checkUnitState(name, js, maxAttempts, out, &wg, errchan)
	}

	go func() {
		wg.Wait()
		close(errchan)
	}()

	return errchan
}

func (f *FleetTunnel) checkUnitState(name string, js job.JobState, maxAttempts int, out io.Writer, wg *sync.WaitGroup, errchan chan error) {
	defer wg.Done()

	sleep := defaultSleepTime

	if maxAttempts < 1 {
		for {
			if f.assertUnitState(name, js, out) {
				return
			}
			time.Sleep(sleep)
		}
	} else {
		for attempt := 0; attempt < maxAttempts; attempt++ {
			if f.assertUnitState(name, js, out) {
				return
			}
			time.Sleep(sleep)
		}
		errchan <- fmt.Errorf("timed out waiting for unit %s to report state %s", name, js)
	}
}

func (f *FleetTunnel) assertUnitState(name string, js job.JobState, out io.Writer) (ret bool) {
	var state string

	u, err := f.cAPI.Unit(name)
	if err != nil {
		log.Warningf("Error retrieving Unit(%s) from Registry: %v", name, err)
		return
	}
	if u == nil {
		log.Warningf("Unit %s not found", name)
		return
	}

	// If this is a global unit, CurrentState will never be set. Instead, wait for DesiredState.
	if suToGlobal(*u) {
		state = u.DesiredState
	} else {
		state = u.CurrentState
	}

	if job.JobState(state) != js {
		log.Debugf("Waiting for Unit(%s) state(%s) to be %s", name, job.JobState(state), js)
		return
	}

	ret = true
	msg := fmt.Sprintf("Unit %s %s", name, u.CurrentState)

	if u.MachineID != "" {
		ms := f.cachedMachineState(u.MachineID)
		if ms != nil {
			msg = fmt.Sprintf("%s on %s", msg, machineFullLegend(*ms, false))
		}
	}

	fmt.Fprintln(out, msg)
	return
}

func (f *FleetTunnel) machineState(machID string) (*machine.MachineState, error) {
	machines, err := f.cAPI.Machines()
	if err != nil {
		return nil, err
	}
	for _, ms := range machines {
		if ms.ID == machID {
			return &ms, nil
		}
	}
	return nil, nil
}

// cachedMachineState makes a best-effort to retrieve the MachineState of the given machine ID.
// It memoizes MachineState information for the life of a fleetctl invocation.
// Any error encountered retrieving the list of machines is ignored.
func (f *FleetTunnel) cachedMachineState(machID string) (ms *machine.MachineState) {
	if f.machineStates == nil {
		f.machineStates = make(map[string]*machine.MachineState)
		ms, err := f.cAPI.Machines()
		if err != nil {
			return nil
		}
		for i, m := range ms {
			f.machineStates[m.ID] = &ms[i]
		}
	}
	return f.machineStates[machID]
}

// setTargetStateOfUnits ensures that the target state for the given Units is set
// to the given state in the Registry.
// On success, a slice of the Units for which a state change was made is returned.
// Any error encountered is immediately returned (i.e. this is not a transaction).
func (f *FleetTunnel) setTargetStateOfUnits(units []string, state job.JobState) ([]*schema.Unit, error) {
	var triggered []*schema.Unit
	for _, name := range units {
		u, err := f.cAPI.Unit(name)
		if err != nil {
			return nil, maskAny(fmt.Errorf("error retrieving unit %s from registry: %v", name, err))
		} else if u == nil {
			return nil, maskAny(fmt.Errorf("unable to find unit %s", name))
		} else if job.JobState(u.DesiredState) == state {
			log.Debugf("Unit(%s) already %s, skipping.", u.Name, u.DesiredState)
			continue
		}

		log.Debugf("Setting Unit(%s) target state to %s", u.Name, state)
		if err := f.cAPI.SetUnitTargetState(u.Name, string(state)); err != nil {
			return nil, maskAny(err)
		}
		triggered = append(triggered, u)
	}

	return triggered, nil
}

func machineFullLegend(ms machine.MachineState, full bool) string {
	legend := machineIDLegend(ms, full)
	if len(ms.PublicIP) > 0 {
		legend = fmt.Sprintf("%s/%s", legend, ms.PublicIP)
	}
	return legend
}

func machineIDLegend(ms machine.MachineState, full bool) string {
	legend := ms.ID
	if !full {
		legend = fmt.Sprintf("%s...", ms.ShortID())
	}
	return legend
}

func getHTTPClient(config FleetConfig) (client.API, error) {
	endPoint := config.EndPoint
	endpoints := strings.Split(endPoint, ",")
	if len(endpoints) > 1 {
		log.Warningf("multiple endpoints provided but only the first (%s) is used", endpoints[0])
	}

	ep, err := url.Parse(endpoints[0])
	if err != nil {
		return nil, err
	}

	if len(ep.Scheme) == 0 {
		return nil, errors.New("URL scheme undefined")
	}

	tun := config.Tunnel
	tunneling := tun != ""

	dialUnix := ep.Scheme == "unix" || ep.Scheme == "file"

	tunnelFunc := net.Dial
	if tunneling {
		sshClient, err := ssh.NewSSHClient(config.SSHUserName, tun, getChecker(config), true, config.SSHTimeout)
		if err != nil {
			return nil, fmt.Errorf("failed initializing SSH client: %v", err)
		}

		if dialUnix {
			tgt := ep.Path
			tunnelFunc = func(string, string) (net.Conn, error) {
				log.Debugf("Establishing remote fleetctl proxy to %s", tgt)
				cmd := fmt.Sprintf(`fleetctl fd-forward %s`, tgt)
				return ssh.DialCommand(sshClient, cmd)
			}
		} else {
			tunnelFunc = sshClient.Dial
		}
	}

	dialFunc := tunnelFunc
	if dialUnix {
		// This commonly happens if the user misses the leading slash after the scheme.
		// For example, "unix://var/run/fleet.sock" would be parsed as host "var".
		if len(ep.Host) > 0 {
			return nil, fmt.Errorf("unable to connect to host %q with scheme %q", ep.Host, ep.Scheme)
		}

		// The Path field is only used for dialing and should not be used when
		// building any further HTTP requests.
		sockPath := ep.Path
		ep.Path = ""

		// If not tunneling to the unix socket, http.Client will dial it directly.
		// http.Client does not natively support dialing a unix domain socket, so the
		// dial function must be overridden.
		if !tunneling {
			dialFunc = func(string, string) (net.Conn, error) {
				return net.Dial("unix", sockPath)
			}
		}

		// http.Client doesn't support the schemes "unix" or "file", but it
		// is safe to use "http" as dialFunc ignores it anyway.
		ep.Scheme = "http"

		// The Host field is not used for dialing, but will be exposed in debug logs.
		ep.Host = "domain-sock"
	}

	tlsConfig, err := pkg.ReadTLSConfigFiles(config.CAFile, config.CertFile, config.KeyFile)
	if err != nil {
		return nil, err
	}

	trans := pkg.LoggingHTTPTransport{
		Transport: http.Transport{
			Dial:            dialFunc,
			TLSClientConfig: tlsConfig,
		},
	}

	hc := http.Client{
		Transport: &trans,
	}

	return client.NewHTTPClient(&hc, *ep)
}

func getRegistryClient(config FleetConfig) (client.API, error) {
	var dial func(string, string) (net.Conn, error)
	tun := config.Tunnel
	if tun != "" {
		sshClient, err := ssh.NewSSHClient(config.SSHUserName, tun, getChecker(config), false, config.SSHTimeout)
		if err != nil {
			return nil, fmt.Errorf("failed initializing SSH client: %v", err)
		}

		dial = func(network, addr string) (net.Conn, error) {
			tcpaddr, err := net.ResolveTCPAddr(network, addr)
			if err != nil {
				return nil, err
			}
			return sshClient.DialTCP(network, nil, tcpaddr)
		}
	}

	tlsConfig, err := pkg.ReadTLSConfigFiles(config.CAFile, config.CertFile, config.KeyFile)
	if err != nil {
		return nil, err
	}

	trans := &http.Transport{
		Dial:            dial,
		TLSClientConfig: tlsConfig,
	}

	eCfg := etcd.Config{
		Endpoints:               strings.Split(config.EndPoint, ","),
		Transport:               trans,
		HeaderTimeoutPerRequest: config.RequestTimeout,
	}

	eClient, err := etcd.New(eCfg)
	if err != nil {
		return nil, err
	}

	kAPI := etcd.NewKeysAPI(eClient)
	reg := registry.NewEtcdRegistry(kAPI, config.EtcdKeyPrefix)

	/*if msg, ok := checkVersion(reg); !ok {
		stderr(msg)
	}*/

	return &client.RegistryClient{Registry: reg}, nil
}

// getChecker creates and returns a HostKeyChecker, or nil if any error is encountered
func getChecker(config FleetConfig) *ssh.HostKeyChecker {
	if !config.StrictHostKeyChecking {
		return nil
	}

	keyFile := ssh.NewHostKeyFile(config.KnownHostsFile)
	return ssh.NewHostKeyChecker(keyFile)
}

// suToGlobal returns whether or not a schema.Unit refers to a global unit
func suToGlobal(su schema.Unit) bool {
	u := job.Unit{
		Unit: *schema.MapSchemaUnitOptionsToUnitFile(su.Options),
	}
	return u.IsGlobal()
}
