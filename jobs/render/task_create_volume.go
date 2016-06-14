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

package render

import (
	"fmt"
	"path"
	"strconv"

	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/pkg/sdunits"
)

// createVolumeUnit
func createVolumeUnit(t *jobs.Task, vol jobs.Volume, volIndex int, ctx generatorContext) (*sdunits.Unit, error) {
	namePostfix := createVolumeUnitNamePostfix(volIndex)
	containerName := createVolumeUnitContainerName(t, volIndex, ctx)
	image := ctx.Images.CephVolume
	volPrefix := path.Join(fmt.Sprintf("%s/%d", t.FullName(), int(ctx.ScalingGroup)), vol.Path)
	volHostPath := fmt.Sprintf("/media/%s", volPrefix)

	unit := &sdunits.Unit{
		Name:         unitName(t, namePostfix, strconv.Itoa(int(ctx.ScalingGroup))),
		FullName:     unitName(t, namePostfix, strconv.Itoa(int(ctx.ScalingGroup))) + ".service",
		Description:  unitDescription(t, fmt.Sprintf("Volume %d", volIndex), ctx.ScalingGroup),
		Type:         "service",
		Scalable_:    true, //t.group.IsScalable(),
		ScalingGroup: ctx.ScalingGroup,
		ExecOptions:  sdunits.NewExecOptions(),
		FleetOptions: sdunits.NewFleetOptions(),
	}
	execStart, err := createVolumeDockerCmdLine(t, containerName, image, vol, volPrefix, volHostPath, unit.ExecOptions.Environment, ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	unit.ExecOptions.ExecStart = execStart.String()
	unit.ExecOptions.Restart = "always"

	unit.ExecOptions.ExecStartPre = []string{
		fmt.Sprintf("/usr/bin/docker pull %s", image),
		fmt.Sprintf("/bin/sh -c 'test -e %s || mkdir -p %s'", volHostPath, volHostPath),
	}

	// Add commands to stop & cleanup existing docker containers
	unit.ExecOptions.ExecStartPre = append(unit.ExecOptions.ExecStartPre,
		fmt.Sprintf("-/usr/bin/docker stop -t %v %s", unit.ExecOptions.ContainerTimeoutStopSec, containerName),
		fmt.Sprintf("-/usr/bin/docker rm -f %s", containerName),
		"-/home/core/bin/docker-cleanup.sh",
	)
	if ctx.DockerOptions.EnvFile != "" {
		unit.ExecOptions.ExecStartPre = append(unit.ExecOptions.ExecStartPre,
			fmt.Sprintf("/usr/bin/touch %s", ctx.DockerOptions.EnvFile),
		)
	}

	unit.ExecOptions.ExecStop = append(unit.ExecOptions.ExecStop,
		fmt.Sprintf("-/usr/bin/docker stop -t %v %s", unit.ExecOptions.ContainerTimeoutStopSec, containerName),
	)
	unit.ExecOptions.ExecStopPost = append(unit.ExecOptions.ExecStopPost,
		fmt.Sprintf("-/usr/bin/docker rm -f %s", containerName),
	)
	if err := setupInstanceConstraints(t, unit, namePostfix, ctx); err != nil {
		return nil, maskAny(err)
	}

	// Service dependencies
	if requires, err := createVolumeRequires(t, ctx); err != nil {
		return nil, maskAny(err)
	} else {
		unit.ExecOptions.Require(requires...)
	}
	unit.ExecOptions.Require("docker.service")
	// After=...
	if after, err := createVolumeAfter(t, ctx); err != nil {
		return nil, maskAny(err)
	} else {
		unit.ExecOptions.After(after...)
	}

	if err := setupConstraints(t, unit); err != nil {
		return nil, maskAny(err)
	}

	addFleetOptions(t, ctx.FleetOptions, unit)

	return unit, nil
}

// createVolumeAfter creates the `After=` sequence for the volume unit
func createVolumeAfter(t *jobs.Task, ctx generatorContext) ([]string, error) {
	after := append([]string{}, commonAfter...)

	return after, nil
}

// createVolumeRequires creates the `Requires=` sequence for the volume unit
func createVolumeRequires(t *jobs.Task, ctx generatorContext) ([]string, error) {
	requires := append([]string{}, commonRequires...)

	return requires, nil
}

// createVolumeUnitNamePostfix creates the volume unit kind + volume-index
func createVolumeUnitNamePostfix(volIndex int) string {
	return fmt.Sprintf("%s%d", unitKindVolume, volIndex)
}

// createVolumeUnitContainerName creates the name of the docker container that serves a volume with given index
func createVolumeUnitContainerName(t *jobs.Task, volIndex int, ctx generatorContext) string {
	return t.ContainerName(ctx.ScalingGroup) + createVolumeUnitNamePostfix(volIndex)
}
