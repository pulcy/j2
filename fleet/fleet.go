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
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	etcd "github.com/coreos/etcd/client"
	"github.com/coreos/fleet/client"
	"github.com/coreos/fleet/pkg"
	"github.com/coreos/fleet/registry"
	"github.com/coreos/fleet/schema"
	"github.com/coreos/fleet/ssh"
	"github.com/juju/errgo"
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
	cAPI client.API
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
	stdOut, err := f.exec("start", append([]string{"--no-block"}, units...)...)
	if err != nil {
		return "", err
	}

	return stdOut, nil
}

func (f *FleetTunnel) Stop(units ...string) (string, error) {
	stdOut, err := f.exec("stop", units...)
	if err != nil {
		return "", err
	}

	return stdOut, nil
}

func (f *FleetTunnel) Destroy(units ...string) (string, error) {
	stdout := bytes.Buffer{}
	errors := []string{}

	for _, unit := range units {
		err := f.cAPI.DestroyUnit(unit)
		if err != nil {
			// Ignore 'Unit does not exist' error
			if client.IsErrorUnitNotFound(err) {
				continue
			}
			errors = append(errors, fmt.Sprintf("Error destroying units: %v", err))
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
					errors = append(errors, fmt.Sprintf("Error destroying units: %v", err))
					break
				}

				if u == nil {
					break
				}
				time.Sleep(defaultSleepTime)
			}
		}

		stdout.WriteString(fmt.Sprintf("Destroyed %s", unit))
	}

	if len(errors) > 0 {
		return "", maskAny(errgo.New(strings.Join(errors, ", ")))
	}

	return stdout.String(), nil
}

func (f *FleetTunnel) List() ([]string, error) {
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
	stdOut, err := f.exec("list-units", "-fields=unit,active", "-full", "-no-legend")
	if err != nil {
		return StatusMap{}, maskAny(err)
	}

	return newStatusMap(stdOut), nil
}

func (f *FleetTunnel) Cat(unitName string) (string, error) {
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

func (f *FleetTunnel) exec(subcmd string, args ...string) (string, error) {
	params := []string{
		"--request-timeout=10",
		"--strict-host-key-checking=false",
		fmt.Sprintf("--tunnel=%s", f.Tunnel),
		subcmd,
	}
	cmd := exec.Command("fleetctl", append(params, args...)...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", errgo.Newf("%s - %v - subcmd: %s", stderr.String(), err, subcmd)
	}

	return stdout.String(), nil
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
