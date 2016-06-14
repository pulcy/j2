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

	"github.com/pulcy/j2/cluster"
	"github.com/pulcy/j2/pkg/sdunits"
)

const (
	unitKindMain   = "-mn"
	unitKindVolume = "-vl"
	unitKindProxy  = "-pr"
	unitKindTimer  = "-ti"
)

var (
	commonAfter = []string{
		"docker.service",
	}
	commonRequires = []string{
		"docker.service",
	}
)

type Task struct {
	Name  TaskName   `json:"name", maspstructure:"-"`
	group *TaskGroup `json:"-", mapstructure:"-"`

	Type             TaskType          `json:"type,omitempty" mapstructure:"type,omitempty"`
	Timer            string            `json:"timer,omitempty" mapstructure:"timer,omitempty"`
	Image            DockerImage       `json:"image"`
	After            []TaskName        `json:"after,omitempty"`
	VolumesFrom      []TaskName        `json:"volumes-from,omitempty"`
	Volumes          VolumeList        `json:"volumes,omitempty"`
	Args             []string          `json:"args,omitempty"`
	Environment      map[string]string `json:"environment,omitempty"`
	Ports            []string          `json:"ports,omitempty"`
	PublicFrontEnds  []PublicFrontEnd  `json:"frontends,omitempty"`
	PrivateFrontEnds []PrivateFrontEnd `json:"private-frontends,omitempty"`
	HttpCheckPath    string            `json:"http-check-path,omitempty" mapstructure:"http-check-path,omitempty"`
	HttpCheckMethod  string            `json:"http-check-method,omitempty" mapstructure:"http-check-method,omitempty"`
	Sticky           bool              `json:"sticky,omitempty" mapstructure:"sticky,omitempty"`
	Capabilities     []string          `json:"capabilities,omitempty"`
	Links            Links             `json:"links,omitempty"`
	Secrets          []Secret          `json:"secrets,omitempty"`
	DockerArgs       []string          `json:"docker-args,omitempty" mapstructure:"docker-args,omitempty"`
	LogDriver        LogDriver         `json:"log-driver,omitempty" mapstructure:"log-driver,omitempty"`
	Target           LinkName          `json:"target,omitempty" mapstructure:"target,omitempty"`
	Rewrites         []Rewrite         `json:"rewrites,omitempty" mapstructure:"rewrites,omitempty"`
	User             string            `json:"user,omitempty" mapstructure:"user,omitempty"`
}

// Link objects just after parsing
func (t *Task) link() {
	t.Target = t.resolveLink(t.Target)
	for i, l := range t.Links {
		t.Links[i].Target = t.resolveLink(l.Target)
	}
	sort.Sort(t.Volumes)
}

// optimizeFor optimizes the task for the given cluster.
func (t *Task) optimizeFor(cluster cluster.Cluster) {
}

// replaceVariables replaces all known variables in the values of the given task.
func (t *Task) replaceVariables() error {
	ctx := NewVariableContext(t.group.job, t.group, t)
	t.Type = TaskType(ctx.replaceString(string(t.Type)))
	t.Timer = ctx.replaceString(t.Timer)
	t.Image = t.Image.replaceVariables(ctx)
	for i, x := range t.After {
		t.After[i] = TaskName(ctx.replaceString(string(x)))
	}
	for i, x := range t.VolumesFrom {
		t.VolumesFrom[i] = TaskName(ctx.replaceString(string(x)))
	}
	for i, x := range t.Volumes {
		t.Volumes[i] = x.replaceVariables(ctx)
	}
	t.Args = ctx.replaceStringSlice(t.Args)
	t.Environment = ctx.replaceStringMap(t.Environment)
	t.Ports = ctx.replaceStringSlice(t.Ports)
	for i, x := range t.PublicFrontEnds {
		t.PublicFrontEnds[i] = x.replaceVariables(ctx)
	}
	for i, x := range t.PrivateFrontEnds {
		t.PrivateFrontEnds[i] = x.replaceVariables(ctx)
	}
	t.HttpCheckPath = ctx.replaceString(t.HttpCheckPath)
	t.HttpCheckMethod = ctx.replaceString(t.HttpCheckMethod)
	t.Capabilities = ctx.replaceStringSlice(t.Capabilities)
	for i, x := range t.Links {
		t.Links[i] = x.replaceVariables(ctx)
	}
	for i, x := range t.Secrets {
		t.Secrets[i] = x.replaceVariables(ctx)
	}
	t.DockerArgs = ctx.replaceStringSlice(t.DockerArgs)
	t.LogDriver = LogDriver(ctx.replaceString(string(t.LogDriver)))
	t.Target = LinkName(ctx.replaceString(string(t.Target)))
	t.User = ctx.replaceString(t.User)
	return maskAny(ctx.Err())
}

// Check for errors
func (t Task) Validate() error {
	if err := t.Name.Validate(); err != nil {
		return maskAny(err)
	}
	if err := t.Type.Validate(); err != nil {
		return maskAny(err)
	}
	for _, name := range t.After {
		_, err := t.group.Task(name)
		if err != nil {
			return maskAny(err)
		}
	}
	for _, name := range t.VolumesFrom {
		_, err := t.group.Task(name)
		if err != nil {
			return maskAny(err)
		}
	}
	for _, l := range t.Links {
		if err := l.Validate(); err != nil {
			return maskAny(err)
		}
	}
	httpFrontends := 0
	tcpFrontends := 0
	for _, f := range t.PublicFrontEnds {
		if err := f.Validate(); err != nil {
			return maskAny(err)
		}
		httpFrontends++
	}
	for _, f := range t.PrivateFrontEnds {
		if err := f.Validate(); err != nil {
			return maskAny(err)
		}
		if f.IsTcp() {
			tcpFrontends++
		} else {
			httpFrontends++
		}
	}
	if tcpFrontends > 0 && httpFrontends > 0 {
		return maskAny(errgo.WithCausef(nil, ValidationError, "cannot mix http and tcp frontends (in '%s')", t.Name))
	}
	for _, s := range t.Secrets {
		if err := s.Validate(); err != nil {
			return maskAny(err)
		}
	}
	if t.Timer != "" {
		if t.Type != "oneshot" {
			return maskAny(errgo.WithCausef(nil, ValidationError, "timer only valid in combination with oneshot (in '%s')", t.Name))
		}
	}
	if err := t.LogDriver.Validate(); err != nil {
		return maskAny(err)
	}
	if t.Target != "" {
		if t.Type != "proxy" {
			return maskAny(errgo.WithCausef(nil, ValidationError, "target only valid in combination with proxy (in '%s')", t.Name))
		}
	}
	if t.Type == "proxy" && t.Target == "" {
		return maskAny(errgo.WithCausef(nil, ValidationError, "target must be set with type proxy (in '%s')", t.Name))
	}
	return nil
}

// createUnits creates all units needed to run this task.
func (t *Task) createUnits(ctx generatorContext) ([]sdunits.UnitChain, error) {
	mainChain := sdunits.UnitChain{}

	sidekickUnitNames := []string{}
	for _, l := range t.Links {
		if !l.Type.IsTCP() {
			continue
		}
		linkIndex := len(sidekickUnitNames)
		unit, err := t.createProxyUnit(l, linkIndex, ctx)
		if err != nil {
			return nil, maskAny(err)
		}
		sidekickUnitNames = append(sidekickUnitNames, unit.FullName)
		mainChain = append(mainChain, unit)
	}

	for i, v := range t.Volumes {
		if !v.requiresMountUnit() {
			continue
		}
		unit, err := t.createVolumeUnit(v, i, ctx)
		if err != nil {
			return nil, maskAny(err)
		}
		sidekickUnitNames = append(sidekickUnitNames, unit.FullName)
		mainChain = append(mainChain, unit)
	}

	main, err := t.createMainUnit(sidekickUnitNames, ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	mainChain = append(mainChain, main)

	timer, err := t.createTimerUnit(ctx)
	if err != nil {
		return nil, maskAny(err)
	}

	chains := []sdunits.UnitChain{mainChain}
	if timer != nil {
		timerChain := sdunits.UnitChain{timer}
		chains = append(chains, timerChain)
	}

	return chains, nil
}

// Gets the full name of this task: job/taskgroup/task
func (t *Task) fullName() string {
	return fmt.Sprintf("%s/%s", t.group.fullName(), t.Name)
}

// privateDomainName returns the DNS name (in the private namespace) for the given task.
func (t *Task) privateDomainName() string {
	ln := NewLinkName(t.group.job.Name, t.group.Name, t.Name, "")
	return ln.PrivateDomainName()
}

// instanceSpecificPrivateDomainName returns the DNS name (in the private namespace) for an instance of the given task.
func (t *Task) instanceSpecificPrivateDomainName(scalingGroup uint) string {
	ln := NewLinkName(t.group.job.Name, t.group.Name, t.Name, InstanceName(strconv.Itoa(int(scalingGroup))))
	return ln.PrivateDomainName()
}

// unitName returns the name of the systemd unit for this task.
func (t *Task) unitName(kind string, scalingGroup string) string {
	base := strings.Replace(t.fullName(), "/", "-", -1) + kind
	if t.group.Global && t.group.Count == 1 {
		return base
	}
	return fmt.Sprintf("%s@%s", base, scalingGroup)
}

// unitDescription creates the description of a unit
func (t *Task) unitDescription(prefix string, scalingGroup uint) string {
	descriptionPostfix := fmt.Sprintf("[slice %d]", scalingGroup)
	if t.group.Global {
		descriptionPostfix = "[global]"
	}
	return fmt.Sprintf("%s unit for %s %s", prefix, t.fullName(), descriptionPostfix)
}

// containerName returns the name of the docker contained used for this task.
func (t *Task) containerName(scalingGroup uint) string {
	return t.containerNameExt(strconv.Itoa(int(scalingGroup)))
}

// containerName returns the name of the docker contained used for this task.
func (t *Task) containerNameExt(instance string) string {
	base := strings.Replace(t.fullName(), "/", "-", -1)
	if t.group.Global {
		return base
	}
	return fmt.Sprintf("%s-%s", base, instance)
}

// serviceName returns the name used to register this service.
func (t *Task) serviceName() string {
	return strings.Replace(t.fullName(), "/", "-", -1)
}

// hasEnvironmentSecrets returns true if the given task has secrets that should
// be stored in an environment variable. False otherwise.
func (t *Task) hasEnvironmentSecrets() bool {
	for _, secret := range t.Secrets {
		if ok, _ := secret.TargetEnviroment(); ok {
			return true
		}
	}
	return false
}

// resolveLink resolves the given (partial) linkname in the context of the given task.
func (t *Task) resolveLink(ln LinkName) LinkName {
	if ln == "" {
		return ln
	}
	jn, tgn, tn, in, _ := ln.parse()
	if jn == "" {
		jn = t.group.job.Name
	}
	if tgn == "" {
		tgn = t.group.Name
	}
	if tn == "" {
		tn = t.Name
	}
	return NewLinkName(jn, tgn, tn, in)
}
