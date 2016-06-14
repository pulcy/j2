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
	"sort"
	"strconv"
	"strings"

	"github.com/juju/errgo"

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
	main.ExecOptions.ExecStart = strings.Join(execStart, " ")
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

// createMainDockerCmdLine creates the `ExecStart` line for
// the main unit.
func createMainDockerCmdLine(t *jobs.Task, image string, env map[string]string, ctx generatorContext) ([]string, error) {
	serviceName := t.ServiceName()
	execStart := []string{
		"/usr/bin/docker",
		"run",
		"--rm",
		fmt.Sprintf("--name %s", t.ContainerName(ctx.ScalingGroup)),
	}
	if len(t.Ports) > 0 {
		for _, p := range t.Ports {
			addArg(fmt.Sprintf("-p %s", p), &execStart, env)
		}
	} else {
		execStart = append(execStart, "-P")
	}
	for i, v := range t.Volumes {
		if v.IsLocal() {
			addArg(fmt.Sprintf("-v %s", v), &execStart, env)
		} else if !v.IsLocal() {
			addArg(fmt.Sprintf("--volumes-from %s", createVolumeUnitContainerName(t, i, ctx)), &execStart, env)
		}
	}
	for _, secret := range t.Secrets {
		if ok, path := secret.TargetFile(); ok {
			hostPath, err := secretFilePath(t, ctx.ScalingGroup, secret)
			if err != nil {
				return nil, maskAny(err)
			}
			addArg(fmt.Sprintf("-v %s:%s:ro", hostPath, path), &execStart, env)
		}
	}
	for _, name := range t.VolumesFrom {
		other, err := t.Task(name)
		if err != nil {
			return nil, maskAny(err)
		}
		for i, v := range other.Volumes {
			if !v.IsLocal() {
				addArg(fmt.Sprintf("--volumes-from %s", createVolumeUnitContainerName(other, i, ctx)), &execStart, env)
			}
		}
		addArg(fmt.Sprintf("--volumes-from %s", other.ContainerName(ctx.ScalingGroup)), &execStart, env)
	}
	envKeys := []string{}
	for k := range t.Environment {
		envKeys = append(envKeys, k)
	}
	sort.Strings(envKeys)
	if ctx.DockerOptions.EnvFile != "" {
		addArg(fmt.Sprintf("--env-file=%s", ctx.DockerOptions.EnvFile), &execStart, env)
	}
	for _, k := range envKeys {
		addArg("-e "+strconv.Quote(fmt.Sprintf("%s=%s", k, t.Environment[k])), &execStart, env)
	}
	if hasEnvironmentSecrets(t) {
		addArg("--env-file="+secretEnvironmentPath(t, ctx.ScalingGroup), &execStart, env)
	}
	addArg(fmt.Sprintf("-e SERVICE_NAME=%s", serviceName), &execStart, env) // Support registrator
	for _, cap := range t.Capabilities {
		addArg("--cap-add "+cap, &execStart, env)
	}
	tcpLinkIndex := 0
	for _, l := range t.Links {
		targetName := l.Target.PrivateDomainName()
		if l.Type.IsHTTP() {
			addArg("--add-host", &execStart, env)
			addArg(fmt.Sprintf("%s:${COREOS_PRIVATE_IPV4}", targetName), &execStart, env)
		} else {
			linkContainerName := fmt.Sprintf("%s-pr%d", t.ContainerName(ctx.ScalingGroup), tcpLinkIndex)
			addArg(fmt.Sprintf("--link %s:%s", linkContainerName, targetName), &execStart, env)
			tcpLinkIndex++
		}
	}
	for _, arg := range t.LogDriver.CreateDockerLogArgs(ctx.DockerOptions) {
		addArg(arg, &execStart, env)
	}
	for _, arg := range t.DockerArgs {
		addArg(arg, &execStart, env)
	}
	if t.User != "" {
		addArg(fmt.Sprintf("--user %s", t.User), &execStart, env)
	}

	execStart = append(execStart, image)
	if t.Type == "proxy" {
		execStart = append(execStart, "sleep 36500d")
	}
	for _, arg := range t.Args {
		addArg(arg, &execStart, env)
	}

	return execStart, nil
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

func setupInstanceConstraints(t *jobs.Task, unit *sdunits.Unit, unitKind string, ctx generatorContext) error {
	unit.FleetOptions.IsGlobal = t.GroupGlobal()
	if ctx.InstanceCount > 1 {
		if t.GroupGlobal() {
			if t.GroupCount() > 1 {
				// Setup metadata constraint such that instances are only scheduled on some machines
				if int(t.GroupCount()) > len(ctx.FleetOptions.GlobalInstanceConstraints) {
					// Group count to high
					return maskAny(errgo.WithCausef(nil, ValidationError, "Group count (%d) higher than #global instance constraints (%d)", t.GroupCount(), len(ctx.FleetOptions.GlobalInstanceConstraints)))
				}
				constraint := ctx.FleetOptions.GlobalInstanceConstraints[ctx.ScalingGroup-1]
				unit.FleetOptions.MachineMetadata(constraint)
			}
		} else {
			unit.FleetOptions.Conflicts(unitName(t, unitKind, "*") + ".service")
		}
	}
	return nil
}

// setupConstraints creates constraint keys for the `X-Fleet` section for the main unit
func setupConstraints(t *jobs.Task, unit *sdunits.Unit) error {
	constraints := t.MergedConstraints()

	metadata := []string{}
	for _, c := range constraints {
		if strings.HasPrefix(c.Attribute, jobs.MetaAttributePrefix) {
			// meta.<somekey>
			key := c.Attribute[len(jobs.MetaAttributePrefix):]
			metadata = append(metadata, fmt.Sprintf("%s=%s", key, c.Value))
		} else {
			switch c.Attribute {
			case jobs.AttributeNodeID:
				unit.FleetOptions.MachineID = c.Value
			default:
				return errgo.WithCausef(nil, ValidationError, "Unknown constraint attribute '%s'", c.Attribute)
			}
		}
	}
	unit.FleetOptions.MachineMetadata(metadata...)

	return nil
}
