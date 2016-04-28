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
	"path"
	"strconv"
	"strings"

	"github.com/pulcy/j2/units"
)

// createVolumeUnit
func (t *Task) createVolumeUnit(vol Volume, volIndex int, ctx generatorContext) (*units.Unit, error) {
	namePostfix := t.createVolumeUnitNamePostfix(volIndex)
	containerName := t.createVolumeUnitContainerName(volIndex, ctx)
	image := ctx.Images.CephVolume
	volPrefix := path.Join(fmt.Sprintf("%s/%d", t.fullName(), int(ctx.ScalingGroup)), vol.Path)
	volHostPath := fmt.Sprintf("/media/%s", volPrefix)

	unit := &units.Unit{
		Name:         t.unitName(namePostfix, strconv.Itoa(int(ctx.ScalingGroup))),
		FullName:     t.unitName(namePostfix, strconv.Itoa(int(ctx.ScalingGroup))) + ".service",
		Description:  t.unitDescription(fmt.Sprintf("Volume %d", volIndex), ctx.ScalingGroup),
		Type:         "service",
		Scalable_:    true, //t.group.IsScalable(),
		ScalingGroup: ctx.ScalingGroup,
		ExecOptions:  units.NewExecOptions(),
		FleetOptions: units.NewFleetOptions(),
	}
	execStart, err := t.createVolumeDockerCmdLine(containerName, image, vol, volPrefix, volHostPath, unit.ExecOptions.Environment, ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	unit.ExecOptions.ExecStart = strings.Join(execStart, " ")
	unit.ExecOptions.Restart = "always"

	unit.ExecOptions.ExecStartPre = []string{
		fmt.Sprintf("/usr/bin/docker pull %s", image),
		fmt.Sprintf("/bin/sh -c 'test -e %s || mkdir -p %s'", volHostPath, volHostPath),
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
	if requires, err := t.createVolumeRequires(ctx); err != nil {
		return nil, maskAny(err)
	} else {
		unit.ExecOptions.Require(requires...)
	}
	unit.ExecOptions.Require("docker.service")
	// After=...
	if after, err := t.createVolumeAfter(ctx); err != nil {
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

// createVolumeDockerCmdLine creates the `ExecStart` line for
// the volume unit.
func (t *Task) createVolumeDockerCmdLine(containerName, containerImage string, vol Volume, volPrefix, volHostPath string, env map[string]string, ctx generatorContext) ([]string, error) {
	execStart := []string{
		"/usr/bin/docker",
		"run",
		"--rm",
		fmt.Sprintf("--name %s", containerName),
		"--net=host",
		"--privileged",
	}
	addArg(fmt.Sprintf("-v %s:%s:shared", volHostPath, vol.Path), &execStart, env)
	addArg("-v /usr/bin/etcdctl:/usr/bin/etcdctl", &execStart, env)
	addArg("-e SERVICE_IGNORE=true", &execStart, env) // Support registrator
	addArg("-e PREFIX="+volPrefix, &execStart, env)
	addArg("-e TARGET="+vol.Path, &execStart, env)
	addArg("-e WAIT=1", &execStart, env)
	if v, err := vol.MountOption("uid"); err == nil {
		addArg("-e UID="+v, &execStart, env)
	}
	if v, err := vol.MountOption("gid"); err == nil {
		addArg("-e GID="+v, &execStart, env)
	}
	for _, arg := range t.LogDriver.CreateDockerLogArgs(ctx.DockerOptions) {
		addArg(arg, &execStart, env)
	}

	execStart = append(execStart, containerImage)

	return execStart, nil
}

// createVolumeAfter creates the `After=` sequence for the volume unit
func (t *Task) createVolumeAfter(ctx generatorContext) ([]string, error) {
	after := append([]string{}, commonAfter...)

	return after, nil
}

// createVolumeRequires creates the `Requires=` sequence for the volume unit
func (t *Task) createVolumeRequires(ctx generatorContext) ([]string, error) {
	requires := append([]string{}, commonRequires...)

	return requires, nil
}

// createVolumeUnitNamePostfix creates the volume unit kind + volume-index
func (t *Task) createVolumeUnitNamePostfix(volIndex int) string {
	return fmt.Sprintf("%s%d", unitKindVolume, volIndex)
}

// createVolumeUnitContainerName creates the name of the docker container that serves a volume with given index
func (t *Task) createVolumeUnitContainerName(volIndex int, ctx generatorContext) string {
	return t.containerName(ctx.ScalingGroup) + t.createVolumeUnitNamePostfix(volIndex)
}
