package fleet

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/juju/errgo"
)

type FleetTunnel struct {
	tunnel string
}

func NewTunnel(tunnel string) *FleetTunnel {
	return &FleetTunnel{
		tunnel: tunnel,
	}
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
	stdOut, err := f.exec("destroy", units...)
	if err != nil {
		return "", err
	}

	return stdOut, nil
}

func (f *FleetTunnel) List() ([]string, error) {
	stdOut, err := f.exec("list-unit-files", "-fields=unit", "-full", "-no-legend")
	if err != nil {
		return []string{}, err
	}

	return strings.Split(strings.TrimSpace(stdOut), "\n"), nil
}

func (f *FleetTunnel) Cat(unitName string) (string, error) {
	stdOut, err := f.exec("cat", unitName)
	if err != nil {
		return "", err
	}

	return stdOut, nil
}

func (f *FleetTunnel) exec(subcmd string, args ...string) (string, error) {
	params := []string{
		"--request-timeout=10",
		"--strict-host-key-checking=false",
		fmt.Sprintf("--tunnel=%s", f.tunnel),
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
