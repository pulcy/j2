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
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	etcd "github.com/coreos/etcd/client"
	"github.com/coreos/fleet/api"
	"github.com/coreos/fleet/client"
	"github.com/coreos/fleet/job"
	"github.com/coreos/fleet/machine"
	"github.com/coreos/fleet/pkg"
	"github.com/coreos/fleet/registry"
	"github.com/coreos/fleet/schema"
	"github.com/coreos/fleet/ssh"
	"github.com/coreos/fleet/unit"
	aerr "github.com/ewoutp/go-aggregate-error"
	"github.com/op/go-logging"
)

const (
	defaultSSHUserName           = "core"
	defaultSSHTimeout            = time.Duration(10.0) * time.Second
	defaultStrictHostKeyChecking = true
	defaultEndpoint              = "unix:///var/run/fleet.sock"
	defaultRegistryEndpoint      = "http://127.0.0.1:2379,http://127.0.0.1:4001"
	defaultRequestTimeout        = time.Duration(3) * time.Second
	defaultSleepTime             = 500 * time.Millisecond
)

var (
	log = logging.MustGetLogger("fleet")
)

type FleetConfig struct {
	Tunnel                string
	SSHUserName           string
	SSHTimeout            time.Duration
	StrictHostKeyChecking bool
	KnownHostsFile        string
	CAFile                string
	CertFile              string
	KeyFile               string
	EndPoint              string
	RequestTimeout        time.Duration
	EtcdKeyPrefix         string
	NoBlock               bool
	BlockAttempts         int
}

type FleetTunnel struct {
	FleetConfig
	cAPI          client.API
	machineStates map[string]*machine.MachineState
}

func NewTunnel(tunnel string) (*FleetTunnel, error) {
	if tunnel != "" && !strings.Contains(tunnel, ":") {
		tunnel += ":22"
	}
	config := FleetConfig{
		Tunnel:                tunnel,
		SSHUserName:           defaultSSHUserName,
		SSHTimeout:            defaultSSHTimeout,
		StrictHostKeyChecking: defaultStrictHostKeyChecking,
		KnownHostsFile:        ssh.DefaultKnownHostsFile,
		CAFile:                "",
		CertFile:              "",
		KeyFile:               "",
		EndPoint:              defaultEndpoint,
		RequestTimeout:        defaultRequestTimeout,
		EtcdKeyPrefix:         registry.DefaultKeyPrefix,
		NoBlock:               false,
		BlockAttempts:         0,
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

func (f *FleetTunnel) Start(units ...string) (string, error) {
	log.Debugf("starting %v", units)

	stdout := &bytes.Buffer{}

	if err := f.lazyCreateUnits(units, stdout); err != nil {
		return "", maskAny(fmt.Errorf("Error creating units: %v", err))
	}

	triggered, err := f.lazyStartUnits(units)
	if err != nil {
		return "", maskAny(fmt.Errorf("Error starting units: %v", err))
	}

	var starting []string
	for _, u := range triggered {
		if suToGlobal(*u) {
			stdout.WriteString(fmt.Sprintf("Triggered global unit %s start\n", u.Name))
		} else {
			starting = append(starting, u.Name)
		}
	}

	if err := f.tryWaitForUnitStates(starting, "start", job.JobStateLaunched, f.BlockAttempts, stdout); err != nil {
		return "", maskAny(err)
	}
	return stdout.String(), nil
}

func (f *FleetTunnel) Stop(unitNames ...string) (string, error) {
	log.Debugf("stopping %v", unitNames)

	stdout := &bytes.Buffer{}

	units, err := f.findUnits(unitNames)
	if err != nil {
		return "", maskAny(err)
	}

	stopping := make([]string, 0)
	for _, u := range units {
		if !suToGlobal(u) {
			if job.JobState(u.CurrentState) == job.JobStateInactive {
				return "", maskAny(fmt.Errorf("Unable to stop unit %s in state %s", u.Name, job.JobStateInactive))
			} else if job.JobState(u.CurrentState) == job.JobStateLoaded {
				log.Debugf("Unit(%s) already %s, skipping.", u.Name, job.JobStateLoaded)
				continue
			}
		}

		log.Debugf("Setting target state of Unit(%s) to %s", u.Name, job.JobStateLoaded)
		f.cAPI.SetUnitTargetState(u.Name, string(job.JobStateLoaded))
		if suToGlobal(u) {
			stdout.WriteString(fmt.Sprintf("Triggered global unit %s stop\n", u.Name))
		} else {
			stopping = append(stopping, u.Name)
		}
	}

	if err := f.tryWaitForUnitStates(stopping, "stop", job.JobStateLoaded, f.BlockAttempts, stdout); err != nil {
		return "", maskAny(err)
	}
	stdout.WriteString(fmt.Sprintf("Successfully stopped units %v.\n", stopping))

	return stdout.String(), nil
}

func (f *FleetTunnel) Destroy(unitNames ...string) (string, error) {
	log.Debugf("destroying %v", unitNames)

	stdout := bytes.Buffer{}
	var ae aerr.AggregateError

	for _, unit := range unitNames {
		err := f.cAPI.DestroyUnit(unit)
		if err != nil {
			// Ignore 'Unit does not exist' error
			if client.IsErrorUnitNotFound(err) {
				continue
			}
			ae.Add(maskAny(fmt.Errorf("Error destroying units: %v", err)))
			continue
		}

		if f.NoBlock {
			attempts := f.BlockAttempts
			retry := func() bool {
				if f.BlockAttempts < 1 {
					return true
				}
				attempts--
				if attempts == 0 {
					return false
				}
				return true
			}

			for retry() {
				u, err := f.cAPI.Unit(unit)
				if err != nil {
					ae.Add(maskAny(fmt.Errorf("Error destroying units: %v", err)))
					break
				}

				if u == nil {
					break
				}
				time.Sleep(defaultSleepTime)
			}
		}

		stdout.WriteString(fmt.Sprintf("Destroyed %s\n", unit))
	}

	if !ae.IsEmpty() {
		return "", maskAny(&ae)
	}

	return stdout.String(), nil
}

func (f *FleetTunnel) List() ([]string, error) {
	log.Debugf("list units")

	units, err := f.cAPI.Units()
	if err != nil {
		return nil, maskAny(err)
	}

	names := []string{}
	for _, unit := range units {
		names = append(names, unit.Name)
	}

	return names, nil
}

func (f *FleetTunnel) Status() (StatusMap, error) {
	log.Debugf("list unit status")

	states, err := f.cAPI.UnitStates()
	if err != nil {
		return StatusMap{}, maskAny(err)
	}

	return newStatusMapFromUnits(states), nil
}

func (f *FleetTunnel) Cat(unitName string) (string, error) {
	log.Debugf("cat unit %v", unitName)

	u, err := f.cAPI.Unit(unitName)
	if err != nil {
		return "", maskAny(err)
	}
	if u == nil {
		return "", maskAny(fmt.Errorf("Unit %s not found", unitName))
	}

	uf := schema.MapSchemaUnitOptionsToUnitFile(u.Options)

	return uf.String(), nil
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

// lazyCreateUnits iterates over a set of unit names and, for each, attempts to
// ensure that a unit by that name exists in the Registry, by checking a number
// of conditions and acting on the first one that succeeds, in order of:
//  1. a unit by that name already existing in the Registry
//  2. a unit file by that name existing on disk
//  3. a corresponding unit template (if applicable) existing in the Registry
//  4. a corresponding unit template (if applicable) existing on disk
// Any error encountered during these steps is returned immediately (i.e.
// subsequent Jobs are not acted on). An error is also returned if none of the
// above conditions match a given Job.
func (f *FleetTunnel) lazyCreateUnits(args []string, stdout io.Writer) error {
	errchan := make(chan error)
	blockAttempts := f.BlockAttempts
	var wg sync.WaitGroup
	for _, arg := range args {
		name := path.Base(arg)

		create, err := f.checkUnitCreation(arg)
		if err != nil {
			return err
		} else if !create {
			continue
		}

		// Assume that the name references a local unit file on
		// disk or if it is an instance unit and if so get its
		// corresponding unit
		uf, err := getUnitFile(arg)
		if err != nil {
			return err
		}

		_, err = f.createUnit(name, uf)
		if err != nil {
			return err
		}

		wg.Add(1)
		go f.checkUnitState(name, job.JobStateInactive, blockAttempts, stdout, &wg, errchan)
	}

	go func() {
		wg.Wait()
		close(errchan)
	}()

	var ae aerr.AggregateError
	for msg := range errchan {
		ae.Add(maskAny(fmt.Errorf("Error waiting on unit creation: %v\n", msg)))
	}

	if !ae.IsEmpty() {
		return maskAny(&ae)
	}

	return nil
}

// checkUnitCreation checks if the unit should be created.
// It takes a unit file path as a parameter.
// It returns 0 on success and if the unit should be created, 1 if the
// unit should not be created; and any error encountered.
func (f *FleetTunnel) checkUnitCreation(arg string) (bool, error) {
	name := path.Base(arg)

	// First, check if there already exists a Unit by the given name in the Registry
	unit, err := f.cAPI.Unit(name)
	if err != nil {
		return false, maskAny(fmt.Errorf("error retrieving Unit(%s) from Registry: %v", name, err))
	}

	// check if the unit is running
	if unit == nil {
		return true, nil
	}
	return false, nil
}

func (f *FleetTunnel) lazyStartUnits(args []string) ([]*schema.Unit, error) {
	units := make([]string, 0, len(args))
	for _, j := range args {
		units = append(units, path.Base(j))
	}
	return f.setTargetStateOfUnits(units, job.JobStateLaunched)
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

// getUnitFile attempts to get a UnitFile configuration
// It takes a unit file name as a parameter and tries first to lookup
// the unit from the local disk. If it fails, it checks if the provided
// file name may reference an instance of a template unit, if so, it
// tries to get the template configuration either from the registry or
// the local disk.
// It returns a UnitFile configuration or nil; and any error ecountered
func getUnitFile(file string) (*unit.UnitFile, error) {
	var uf *unit.UnitFile
	name := path.Base(file)

	log.Debugf("Looking for Unit(%s) or its corresponding template", name)

	// Assume that the file references a local unit file on disk and
	// attempt to load it, if it exists
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		uf, err = getUnitFromFile(file)
		if err != nil {
			return nil, maskAny(fmt.Errorf("failed getting Unit(%s) from file: %v", file, err))
		}
	} else {
		// Otherwise (if the unit file does not exist), check if the
		// name appears to be an instance of a template unit
		info := unit.NewUnitNameInfo(name)
		if info == nil {
			return nil, maskAny(fmt.Errorf("error extracting information from unit name %s", name))
		} else if !info.IsInstance() {
			return nil, maskAny(fmt.Errorf("unable to find Unit(%s) in Registry or on filesystem", name))
		}

		// If it is an instance check for a corresponding template
		// unit in the Registry or disk.
		// If we found a template unit, later we create a
		// near-identical instance unit in the Registry - same
		// unit file as the template, but different name
		uf, err = getUnitFileFromTemplate(info, file)
		if err != nil {
			return nil, maskAny(fmt.Errorf("failed getting Unit(%s) from template: %v", file, err))
		}
	}

	log.Debugf("Found Unit(%s)", name)
	return uf, nil
}

// getUnitFromFile attempts to load a Unit from a given filename
// It returns the Unit or nil, and any error encountered
func getUnitFromFile(file string) (*unit.UnitFile, error) {
	out, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, maskAny(err)
	}

	unitName := path.Base(file)
	log.Debugf("Unit(%s) found in local filesystem", unitName)

	return unit.NewUnitFile(string(out))
}

// getUnitFileFromTemplate attempts to get a Unit from a template unit that
// is either in the registry or on the file system
// It takes two arguments, the template information and the unit file name
// It returns the Unit or nil; and any error encountered
func getUnitFileFromTemplate(uni *unit.UnitNameInfo, fileName string) (*unit.UnitFile, error) {
	// Load template from disk
	filePath := path.Join(path.Dir(fileName), uni.Template)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, maskAny(fmt.Errorf("unable to find template Unit(%s) in Registry or on filesystem", uni.Template))
	}

	uf, err := getUnitFromFile(filePath)
	if err != nil {
		return nil, maskAny(fmt.Errorf("unable to load template Unit(%s) from file: %v", uni.Template, err))
	}

	return uf, nil
}

func (f *FleetTunnel) createUnit(name string, uf *unit.UnitFile) (*schema.Unit, error) {
	if uf == nil {
		return nil, maskAny(fmt.Errorf("nil unit provided"))
	}
	u := schema.Unit{
		Name:    name,
		Options: schema.MapUnitFileToSchemaUnitOptions(uf),
	}
	// TODO(jonboulle): this dependency on the API package is awkward, and
	// redundant with the check in api.unitsResource.set, but it is a
	// workaround to implementing the same check in the RegistryClient. It
	// will disappear once RegistryClient is deprecated.
	if err := api.ValidateName(name); err != nil {
		return nil, maskAny(err)
	}
	if err := api.ValidateOptions(u.Options); err != nil {
		return nil, maskAny(err)
	}
	j := &job.Job{Unit: *uf}
	if err := j.ValidateRequirements(); err != nil {
		log.Warningf("Unit %s: %v", name, err)
	}
	err := f.cAPI.CreateUnit(&u)
	if err != nil {
		return nil, maskAny(fmt.Errorf("failed creating unit %s: %v", name, err))
	}

	log.Debugf("Created Unit(%s) in Registry", name)
	return &u, nil
}
