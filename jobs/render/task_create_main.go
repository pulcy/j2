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
	"strconv"

	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/pkg/sdunits"
)

// createMainUnit
func createMainUnit(t *jobs.Task, sidekickUnitNames []string, ctx generatorContext) (*sdunits.Unit, error) {
	name := t.ContainerName(ctx.ScalingGroup)
	image := t.Image.String()
	if t.Type == "proxy" {
		image = ctx.Images.Alpine
	}

	main := &sdunits.Unit{
		Name:         unitName(t, unitKindMain, strconv.Itoa(int(ctx.ScalingGroup))),
		FullName:     unitName(t, unitKindMain, strconv.Itoa(int(ctx.ScalingGroup))) + ".service",
		Description:  unitDescription(t, "Main", ctx.ScalingGroup),
		Type:         "service",
		Scalable_:    true, //t.group.IsScalable(),
		ScalingGroup: ctx.ScalingGroup,
		ExecOptions:  sdunits.NewExecOptions(),
		FleetOptions: sdunits.NewFleetOptions(),
	}
	execStart, err := createMainDockerCmdLine(t, image, main.ExecOptions.Environment, ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	main.ExecOptions.ExecStart = execStart.String()
	switch t.Type {
	case "oneshot":
		main.ExecOptions.IsOneshot = true
		main.ExecOptions.Restart = "on-failure"
	case "proxy":
		main.ExecOptions.Restart = "always"
	default:
		main.ExecOptions.Restart = "always"
	}
	main.ExecOptions.ExecStartPre = []string{
		fmt.Sprintf("/usr/bin/docker pull %s", image),
	}
	if ctx.DockerOptions.EnvFile != "" {
		main.ExecOptions.ExecStartPre = append(main.ExecOptions.ExecStartPre,
			fmt.Sprintf("/usr/bin/touch %s", ctx.DockerOptions.EnvFile),
		)
	}

	// Add secret extraction commands
	secretsCmds, err := createSecretsExecStartPre(t, main.ExecOptions.Environment, ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	main.ExecOptions.ExecStartPre = append(main.ExecOptions.ExecStartPre, secretsCmds...)

	// Add commands to stop & cleanup existing docker containers
	main.ExecOptions.ExecStartPre = append(main.ExecOptions.ExecStartPre,
		fmt.Sprintf("-/usr/bin/docker stop -t %v %s", main.ExecOptions.ContainerTimeoutStopSec, name),
		fmt.Sprintf("-/usr/bin/docker rm -f %s", t.ContainerName(ctx.ScalingGroup)),
		"-/home/core/bin/docker-cleanup.sh",
	)
	for _, v := range t.Volumes {
		if v.IsLocal() {
			hostPath := v.HostPath
			mkdir := fmt.Sprintf("/bin/sh -c 'test -e %s || mkdir -p %s'", hostPath, hostPath)
			main.ExecOptions.ExecStartPre = append(main.ExecOptions.ExecStartPre, mkdir)
		}
	}

	main.ExecOptions.ExecStop = append(main.ExecOptions.ExecStop,
		fmt.Sprintf("-/usr/bin/docker stop -t %v %s", main.ExecOptions.ContainerTimeoutStopSec, name),
	)
	main.ExecOptions.ExecStopPost = append(main.ExecOptions.ExecStopPost,
		fmt.Sprintf("-/usr/bin/docker rm -f %s", name),
	)

	if err := setupInstanceConstraints(t, main, unitKindMain, ctx); err != nil {
		return nil, maskAny(err)
	}

	// Service dependencies
	// Requires=
	if requires, err := createMainRequires(t, sidekickUnitNames, ctx); err != nil {
		return nil, maskAny(err)
	} else {
		main.ExecOptions.Require(requires...)
	}
	main.ExecOptions.Require("docker.service")
	// After=...
	if after, err := createMainAfter(t, sidekickUnitNames, ctx); err != nil {
		return nil, maskAny(err)
	} else {
		main.ExecOptions.After(after...)
	}

	if err := addFrontEndRegistration(t, main, ctx); err != nil {
		return nil, maskAny(err)
	}

	if err := setupConstraints(t, main); err != nil {
		return nil, maskAny(err)
	}

	addFleetOptions(t, ctx.FleetOptions, main)

	return main, nil
}

// createMainAfter creates the `After=` sequence for the main unit
func createMainAfter(t *jobs.Task, sidekickUnitNames []string, ctx generatorContext) ([]string, error) {
	after := append([]string{}, commonAfter...)
	after = append(after, sidekickUnitNames...)

	for _, name := range t.VolumesFrom {
		other, err := t.Task(name)
		if err != nil {
			return nil, maskAny(err)
		}
		after = append(after, unitName(other, unitKindMain, strconv.Itoa(int(ctx.ScalingGroup)))+".service")
	}

	return after, nil
}

// createMainRequires creates the `Requires=` sequence for the main unit
func createMainRequires(t *jobs.Task, sidekickUnitNames []string, ctx generatorContext) ([]string, error) {
	requires := append([]string{}, commonRequires...)
	requires = append(requires, sidekickUnitNames...)

	for _, name := range t.VolumesFrom {
		other, err := t.Task(name)
		if err != nil {
			return nil, maskAny(err)
		}
		requires = append(requires, unitName(other, unitKindMain, strconv.Itoa(int(ctx.ScalingGroup)))+".service")
	}

	return requires, nil
}
