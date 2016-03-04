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
	"sort"
	"strconv"
	"strings"

	"github.com/juju/errgo"

	"github.com/pulcy/j2/units"
)

// createMainUnit
func (t *Task) createMainUnit(proxyUnitNames []string, ctx generatorContext) (*units.Unit, error) {
	name := t.containerName(ctx.ScalingGroup)
	image := t.Image.String()

	main := &units.Unit{
		Name:         t.unitName(unitKindMain, strconv.Itoa(int(ctx.ScalingGroup))),
		FullName:     t.unitName(unitKindMain, strconv.Itoa(int(ctx.ScalingGroup))) + ".service",
		Description:  t.unitDescription("Main", ctx.ScalingGroup),
		Type:         "service",
		Scalable_:    true, //t.group.IsScalable(),
		ScalingGroup: ctx.ScalingGroup,
		ExecOptions:  units.NewExecOptions(),
		FleetOptions: units.NewFleetOptions(),
	}
	execStart, err := t.createMainDockerCmdLine(main.ExecOptions.Environment, ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	main.ExecOptions.ExecStart = strings.Join(execStart, " ")
	switch t.Type {
	case "oneshot":
		main.ExecOptions.IsOneshot = true
		main.ExecOptions.Restart = "on-failure"
	case "proxy":
		main.ExecOptions.IsOneshot = true
		main.ExecOptions.Restart = "on-failure"
		main.ExecOptions.ExecStart = "/usr/bin/true"
	default:
		main.ExecOptions.Restart = "always"
	}
	//main.FleetOptions.IsGlobal = ds.global
	if t.Type != "proxy" {
		main.ExecOptions.ExecStartPre = []string{
			fmt.Sprintf("/usr/bin/docker pull %s", image),
		}
	}

	// Add secret extraction commands
	secretsCmds, err := t.createSecretsExecStartPre(main.ExecOptions.Environment, ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	main.ExecOptions.ExecStartPre = append(main.ExecOptions.ExecStartPre, secretsCmds...)

	// Add commands to stop & cleanup existing docker containers
	if t.Type != "proxy" {
		main.ExecOptions.ExecStartPre = append(main.ExecOptions.ExecStartPre,
			fmt.Sprintf("-/usr/bin/docker stop -t %v %s", main.ExecOptions.ContainerTimeoutStopSec, name),
			fmt.Sprintf("-/usr/bin/docker rm -f %s", t.containerName(ctx.ScalingGroup)),
		)
	}
	for _, v := range t.Volumes {
		dir := strings.Split(v, ":")
		mkdir := fmt.Sprintf("/bin/sh -c 'test -e %s || mkdir -p %s'", dir[0], dir[0])
		main.ExecOptions.ExecStartPre = append(main.ExecOptions.ExecStartPre, mkdir)
	}

	if t.Type != "proxy" {
		main.ExecOptions.ExecStop = append(main.ExecOptions.ExecStop,
			fmt.Sprintf("-/usr/bin/docker stop -t %v %s", main.ExecOptions.ContainerTimeoutStopSec, name),
		)
		main.ExecOptions.ExecStopPost = append(main.ExecOptions.ExecStopPost,
			fmt.Sprintf("-/usr/bin/docker rm -f %s", name),
		)
	}

	if err := t.setupInstanceConstraints(main, unitKindMain, ctx); err != nil {
		return nil, maskAny(err)
	}

	// Service dependencies
	// Requires=
	if requires, err := t.createMainRequires(proxyUnitNames, ctx); err != nil {
		return nil, maskAny(err)
	} else {
		main.ExecOptions.Require(requires...)
	}
	main.ExecOptions.Require("docker.service")
	// After=...
	if after, err := t.createMainAfter(proxyUnitNames, ctx); err != nil {
		return nil, maskAny(err)
	} else {
		main.ExecOptions.After(after...)
	}

	if err := t.addFrontEndRegistration(main, ctx); err != nil {
		return nil, maskAny(err)
	}

	if err := t.setupConstraints(main); err != nil {
		return nil, maskAny(err)
	}

	t.AddFleetOptions(ctx.FleetOptions, main)

	return main, nil
}

// createMainDockerCmdLine creates the `ExecStart` line for
// the main unit.
func (t *Task) createMainDockerCmdLine(env map[string]string, ctx generatorContext) ([]string, error) {
	serviceName := t.serviceName()
	image := t.Image.String()
	execStart := []string{
		"/usr/bin/docker",
		"run",
		"--rm",
		fmt.Sprintf("--name %s", t.containerName(ctx.ScalingGroup)),
	}
	if len(t.Ports) > 0 {
		for _, p := range t.Ports {
			addArg(fmt.Sprintf("-p %s", p), &execStart, env)
		}
	} else {
		execStart = append(execStart, "-P")
	}
	for _, v := range t.Volumes {
		addArg(fmt.Sprintf("-v %s", v), &execStart, env)
	}
	for _, secret := range t.Secrets {
		if ok, path := secret.TargetFile(); ok {
			hostPath, err := t.secretFilePath(ctx.ScalingGroup, secret)
			if err != nil {
				return nil, maskAny(err)
			}
			addArg(fmt.Sprintf("-v %s:%s:ro", hostPath, path), &execStart, env)
		}
	}
	for _, name := range t.VolumesFrom {
		other, err := t.group.Task(name)
		if err != nil {
			return nil, maskAny(err)
		}
		addArg(fmt.Sprintf("--volumes-from %s", other.containerName(ctx.ScalingGroup)), &execStart, env)
	}
	envKeys := []string{}
	for k := range t.Environment {
		envKeys = append(envKeys, k)
	}
	sort.Strings(envKeys)
	for _, k := range envKeys {
		addArg("-e "+strconv.Quote(fmt.Sprintf("%s=%s", k, t.Environment[k])), &execStart, env)
	}
	if t.hasEnvironmentSecrets() {
		addArg("--env-file="+t.secretEnvironmentPath(ctx.ScalingGroup), &execStart, env)
	}
	addArg(fmt.Sprintf("-e SERVICE_NAME=%s", serviceName), &execStart, env) // Support registrator
	for _, cap := range t.Capabilities {
		addArg("--cap-add "+cap, &execStart, env)
	}
	tcpLinkIndex := 0
	for _, l := range t.Links {
		targetName := l.Target.PrivateDomainName()
		if l.Type.IsHTTP() {
			addArg(fmt.Sprintf("--add-host %s:${COREOS_PRIVATE_IPV4}", targetName), &execStart, env)
		} else {
			linkContainerName := fmt.Sprintf("%s-pr%d", t.containerName(ctx.ScalingGroup), tcpLinkIndex)
			addArg(fmt.Sprintf("--link %s:%s", linkContainerName, targetName), &execStart, env)
			tcpLinkIndex++
		}
	}
	for _, arg := range t.LogDriver.CreateDockerLogArgs(ctx.DockerOptions) {
		addArg(arg, &execStart, env)
	}
	execStart = append(execStart, t.DockerArgs...)

	execStart = append(execStart, image)
	execStart = append(execStart, t.Args...)

	return execStart, nil
}

// createMainAfter creates the `After=` sequence for the main unit
func (t *Task) createMainAfter(proxyUnitNames []string, ctx generatorContext) ([]string, error) {
	after := append([]string{}, commonAfter...)
	after = append(after, proxyUnitNames...)

	for _, name := range t.VolumesFrom {
		other, err := t.group.Task(name)
		if err != nil {
			return nil, maskAny(err)
		}
		after = append(after, other.unitName(unitKindMain, strconv.Itoa(int(ctx.ScalingGroup)))+".service")
	}

	return after, nil
}

// createMainRequires creates the `Requires=` sequence for the main unit
func (t *Task) createMainRequires(proxyUnitNames []string, ctx generatorContext) ([]string, error) {
	requires := append([]string{}, commonRequires...)
	requires = append(requires, proxyUnitNames...)

	for _, name := range t.VolumesFrom {
		other, err := t.group.Task(name)
		if err != nil {
			return nil, maskAny(err)
		}
		requires = append(requires, other.unitName(unitKindMain, strconv.Itoa(int(ctx.ScalingGroup)))+".service")
	}

	return requires, nil
}

func (t *Task) setupInstanceConstraints(unit *units.Unit, unitKind string, ctx generatorContext) error {
	unit.FleetOptions.IsGlobal = t.group.Global
	if ctx.InstanceCount > 1 {
		if t.group.Global {
			if t.group.Count > 1 {
				// Setup metadata constraint such that instances are only scheduled on some machines
				if int(t.group.Count) > len(ctx.FleetOptions.GlobalInstanceConstraints) {
					// Group count to high
					return maskAny(errgo.WithCausef(nil, ValidationError, "Group count (%d) higher than #global instance constraints (%d)", t.group.Count, len(ctx.FleetOptions.GlobalInstanceConstraints)))
				}
				constraint := ctx.FleetOptions.GlobalInstanceConstraints[ctx.ScalingGroup-1]
				unit.FleetOptions.MachineMetadata(constraint)
			}
		} else {
			unit.FleetOptions.Conflicts(t.unitName(unitKind, "*") + ".service")
		}
	}
	return nil
}

// setupConstraints creates constraint keys for the `X-Fleet` section for the main unit
func (t *Task) setupConstraints(unit *units.Unit) error {
	constraints := t.group.job.Constraints.Merge(t.group.Constraints)

	metadata := []string{}
	for _, c := range constraints {
		if strings.HasPrefix(c.Attribute, metaAttributePrefix) {
			// meta.<somekey>
			key := c.Attribute[len(metaAttributePrefix):]
			metadata = append(metadata, fmt.Sprintf("%s=%s", key, c.Value))
		} else {
			switch c.Attribute {
			case attributeNodeID:
				unit.FleetOptions.MachineID = c.Value
			default:
				return errgo.WithCausef(nil, ValidationError, "Unknown constraint attribute '%s'", c.Attribute)
			}
		}
	}
	unit.FleetOptions.MachineMetadata(metadata...)

	return nil
}
