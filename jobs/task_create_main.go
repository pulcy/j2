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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/dchest/uniuri"
	"github.com/juju/errgo"
	"github.com/nyarla/go-crypt"

	"github.com/pulcy/j2/units"
)

var (
	FixedPwhashSalt string // If set, this salt will be used for all pwhash's (only used for testing)
)

// createMainUnit
func (t *Task) createMainUnit(ctx generatorContext) (*units.Unit, error) {
	name := t.containerName(ctx.ScalingGroup)
	image := t.Image.String()

	main := &units.Unit{
		Name:         t.unitName(unitKindMain, strconv.Itoa(int(ctx.ScalingGroup))),
		FullName:     t.unitName(unitKindMain, strconv.Itoa(int(ctx.ScalingGroup))) + ".service",
		Description:  t.unitDescription("Main", ctx.ScalingGroup),
		Type:         "service",
		Scalable:     t.group.IsScalable(),
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
	default:
		main.ExecOptions.Restart = "always"
	}
	//main.FleetOptions.IsGlobal = ds.global
	main.ExecOptions.ExecStartPre = []string{
		fmt.Sprintf("/usr/bin/docker pull %s", image),
	}

	// Add secret extraction commands
	secretsCmds, err := t.createSecretsExecStartPre(main.ExecOptions.Environment, ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	main.ExecOptions.ExecStartPre = append(main.ExecOptions.ExecStartPre, secretsCmds...)

	// Add commands to stop & cleanup existing docker containers
	main.ExecOptions.ExecStartPre = append(main.ExecOptions.ExecStartPre,
		fmt.Sprintf("-/usr/bin/docker stop -t %v %s", main.ExecOptions.ContainerTimeoutStopSec, name),
		fmt.Sprintf("-/usr/bin/docker rm -f %s", t.containerName(ctx.ScalingGroup)),
	)
	for _, v := range t.Volumes {
		dir := strings.Split(v, ":")
		mkdir := fmt.Sprintf("/bin/sh -c 'test -e %s || mkdir -p %s'", dir[0], dir[0])
		main.ExecOptions.ExecStartPre = append(main.ExecOptions.ExecStartPre, mkdir)
	}

	main.ExecOptions.ExecStop = append(main.ExecOptions.ExecStop,
		fmt.Sprintf("-/usr/bin/docker stop -t %v %s", main.ExecOptions.ContainerTimeoutStopSec, name),
	)
	main.ExecOptions.ExecStopPost = append(main.ExecOptions.ExecStopPost,
		fmt.Sprintf("-/usr/bin/docker rm -f %s", name),
	)
	main.FleetOptions.IsGlobal = t.group.Global
	if t.group.IsScalable() && ctx.InstanceCount > 1 {
		main.FleetOptions.Conflicts(t.unitName(unitKindMain, "*") + ".service")
	}

	// Service dependencies
	// Requires=
	//main.ExecOptions.Require("flanneld.service")
	if requires, err := t.createMainRequires(ctx); err != nil {
		return nil, maskAny(err)
	} else {
		main.ExecOptions.Require(requires...)
	}
	main.ExecOptions.Require("docker.service")
	// After=...
	if after, err := t.createMainAfter(ctx); err != nil {
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
	for _, ln := range t.Links {
		addArg(fmt.Sprintf("--add-host %s:${COREOS_PRIVATE_IPV4}", ln.PrivateDomainName()), &execStart, env)
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
func (t *Task) createMainAfter(ctx generatorContext) ([]string, error) {
	after := append([]string{}, commonAfter...)

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
func (t *Task) createMainRequires(ctx generatorContext) ([]string, error) {
	requires := append([]string{}, commonRequires...)

	for _, name := range t.VolumesFrom {
		other, err := t.group.Task(name)
		if err != nil {
			return nil, maskAny(err)
		}
		requires = append(requires, other.unitName(unitKindMain, strconv.Itoa(int(ctx.ScalingGroup)))+".service")
	}

	return requires, nil
}

// setupConstraints creates constraint keys for the `X-Fleet` section for the main unit
func (t *Task) setupConstraints(unit *units.Unit) error {
	constraints := t.group.job.Constraints.Merge(t.group.Constraints).Merge(t.Constraints)

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

type frontendRecord struct {
	Selectors     []frontendSelectorRecord `json:"selectors"`
	Service       string                   `json:"service,omitempty"`
	Mode          string                   `json:"mode,omitempty"` // http|tcp
	HttpCheckPath string                   `json:"http-check-path,omitempty"`
}

type frontendSelectorRecord struct {
	Weight     int          `json:"weight,omitempty"`
	Domain     string       `json:"domain,omitempty"`
	PathPrefix string       `json:"path-prefix,omitempty"`
	SslCert    string       `json:"ssl-cert,omitempty"`
	Port       int          `json:"port,omitempty"`
	Private    bool         `json:"private,omitempty"`
	Users      []userRecord `json:"users,omitempty"`
}

type userRecord struct {
	Name         string `json:"user"`
	PasswordHash string `json:"pwhash"`
}

// addFrontEndRegistration adds registration code for frontends to the given units
func (t *Task) addFrontEndRegistration(main *units.Unit, ctx generatorContext) error {
	if len(t.PublicFrontEnds) == 0 && len(t.PrivateFrontEnds) == 0 {
		return nil
	}
	key := fmt.Sprintf("/pulcy/frontend/%s-%d", t.serviceName(), ctx.ScalingGroup)
	record := frontendRecord{
		Service:       t.serviceName(),
		HttpCheckPath: t.HttpCheckPath,
	}
	instanceKey := fmt.Sprintf("/pulcy/frontend/%s-%d-inst", t.serviceName(), ctx.ScalingGroup)
	instanceRecord := frontendRecord{
		Service:       fmt.Sprintf("%s-%d", t.serviceName(), ctx.ScalingGroup),
		HttpCheckPath: t.HttpCheckPath,
	}

	for _, fr := range t.PublicFrontEnds {
		selRecord := frontendSelectorRecord{
			Weight:     fr.Weight,
			Domain:     fr.Domain,
			PathPrefix: fr.PathPrefix,
			SslCert:    fr.SslCert,
			Port:       fr.Port,
		}
		selRecord.addUsers(fr.Users)
		record.Selectors = append(record.Selectors, selRecord)
	}
	for _, fr := range t.PrivateFrontEnds {
		if fr.Mode == "tcp" {
			record.Mode = "tcp"
		}
		selRecord := frontendSelectorRecord{
			Domain:  t.privateDomainName(),
			Port:    fr.Port,
			Private: true,
		}
		selRecord.addUsers(fr.Users)
		record.Selectors = append(record.Selectors, selRecord)

		if fr.RegisterInstance {
			instanceSelRecord := selRecord
			instanceSelRecord.Domain = t.instanceSpecificPrivateDomainName(ctx.ScalingGroup)
			instanceRecord.Selectors = append(instanceRecord.Selectors, instanceSelRecord)
		}
	}

	if len(instanceRecord.Selectors) > 0 {
		if err := t.addFrontEndRegistrationRecord(main, instanceKey, instanceRecord, "FrontEndRegistration-i"); err != nil {
			return maskAny(err)
		}
	}
	if err := t.addFrontEndRegistrationRecord(main, key, record, "FrontEndRegistration"); err != nil {
		return maskAny(err)
	}

	return nil
}

func (t *Task) addFrontEndRegistrationRecord(main *units.Unit, key string, record frontendRecord, projectSettingKey string) error {
	json, err := json.Marshal(&record)
	if err != nil {
		return maskAny(err)
	}
	main.ProjectSetting(projectSettingKey, key+"="+string(json))
	main.ExecOptions.ExecStartPost = append(main.ExecOptions.ExecStartPost,
		fmt.Sprintf("/bin/sh -c 'echo %s | base64 -d | /usr/bin/etcdctl set %s'", base64.StdEncoding.EncodeToString(json), key),
	)
	main.ExecOptions.ExecStop = append(
		[]string{fmt.Sprintf("-/usr/bin/etcdctl rm %s", key)},
		main.ExecOptions.ExecStop...,
	)
	return nil
}

// addUsers adds the given users to the selector record, while encrypting the passwords.
func (selRecord *frontendSelectorRecord) addUsers(users []User) {
	for _, u := range users {
		salt := FixedPwhashSalt
		if salt == "" {
			salt = uniuri.New()
		}
		userRec := userRecord{
			Name:         u.Name,
			PasswordHash: crypt.Crypt(u.Password, salt),
		}
		selRecord.Users = append(selRecord.Users, userRec)
	}
}
