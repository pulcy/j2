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

package jobs

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pulcy/j2/units"
)

// createProxyUnit
func (t *Task) createProxyUnit(link Link, linkIndex int, ctx generatorContext) (*units.Unit, error) {
	namePostfix := fmt.Sprintf("%s%d", unitKindProxy, linkIndex)
	containerName := t.containerName(ctx.ScalingGroup) + namePostfix
	image := ctx.Images.Wormhole

	unit := &units.Unit{
		Name:         t.unitName(namePostfix, strconv.Itoa(int(ctx.ScalingGroup))),
		FullName:     t.unitName(namePostfix, strconv.Itoa(int(ctx.ScalingGroup))) + ".service",
		Description:  t.unitDescription(fmt.Sprintf("Proxy %d", linkIndex), ctx.ScalingGroup),
		Type:         "service",
		Scalable_:    true, //t.group.IsScalable(),
		ScalingGroup: ctx.ScalingGroup,
		ExecOptions:  units.NewExecOptions(),
		FleetOptions: units.NewFleetOptions(),
	}
	execStart, err := t.createProxyDockerCmdLine(containerName, image, link, unit.ExecOptions.Environment, ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	unit.ExecOptions.ExecStart = strings.Join(execStart, " ")
	unit.ExecOptions.Restart = "always"

	unit.ExecOptions.ExecStartPre = []string{
		fmt.Sprintf("/usr/bin/docker pull %s", image),
	}

	// Add commands to stop & cleanup existing docker containers
	unit.ExecOptions.ExecStartPre = append(unit.ExecOptions.ExecStartPre,
		fmt.Sprintf("-/usr/bin/docker stop -t %v %s", unit.ExecOptions.ContainerTimeoutStopSec, containerName),
		fmt.Sprintf("-/usr/bin/docker rm -f %s", containerName),
	)

	unit.ExecOptions.ExecStop = append(unit.ExecOptions.ExecStop,
		fmt.Sprintf("-/usr/bin/docker stop -t %v %s", unit.ExecOptions.ContainerTimeoutStopSec, containerName),
	)
	unit.ExecOptions.ExecStopPost = append(unit.ExecOptions.ExecStopPost,
		fmt.Sprintf("-/usr/bin/docker rm -f %s", containerName),
	)
	if err := t.setupInstanceConstraints(unit, namePostfix, ctx); err != nil {
		return nil, maskAny(err)
	}

	// Service dependencies
	if requires, err := t.createProxyRequires(ctx); err != nil {
		return nil, maskAny(err)
	} else {
		unit.ExecOptions.Require(requires...)
	}
	unit.ExecOptions.Require("docker.service")
	// After=...
	if after, err := t.createProxyAfter(ctx); err != nil {
		return nil, maskAny(err)
	} else {
		unit.ExecOptions.After(after...)
	}

	if err := t.setupConstraints(unit); err != nil {
		return nil, maskAny(err)
	}

	t.AddFleetOptions(ctx.FleetOptions, unit)

	return unit, nil
}

// createProxyDockerCmdLine creates the `ExecStart` line for
// the proxy unit.
func (t *Task) createProxyDockerCmdLine(containerName, containerImage string, link Link, env map[string]string, ctx generatorContext) ([]string, error) {
	execStart := []string{
		"/usr/bin/docker",
		"run",
		"--rm",
		fmt.Sprintf("--name %s", containerName),
	}
	for _, p := range link.Ports {
		addArg(fmt.Sprintf("--expose %d", p), &execStart, env)
	}
	addArg("-P", &execStart, env)
	addArg("-e SERVICE_IGNORE=true", &execStart, env) // Support registrator
	for _, arg := range t.LogDriver.CreateDockerLogArgs(ctx.DockerOptions) {
		addArg(arg, &execStart, env)
	}

	execStart = append(execStart, containerImage)
	execStart = append(execStart,
		fmt.Sprintf("--etcd-addr http://${COREOS_PRIVATE_IPV4}:2379/pulcy/service/%s", link.Target.etcdServiceName()),
	)

	return execStart, nil
}

// createProxyAfter creates the `After=` sequence for the proxy unit
func (t *Task) createProxyAfter(ctx generatorContext) ([]string, error) {
	after := append([]string{}, commonAfter...)

	return after, nil
}

// createProxyRequires creates the `Requires=` sequence for the proxy unit
func (t *Task) createProxyRequires(ctx generatorContext) ([]string, error) {
	requires := append([]string{}, commonRequires...)

	return requires, nil
}
