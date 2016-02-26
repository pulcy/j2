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

	"github.com/juju/errgo"

	"github.com/pulcy/j2/units"
)

const (
	unitKindMain  = "-mn"
	unitKindProxy = "-pr"
	unitKindTimer = "-ti"
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
	Name   TaskName   `json:"name", maspstructure:"-"`
	group  *TaskGroup `json:"-", mapstructure:"-"`
	Count  uint       `json:"-"` // This value is used during parsing only
	Global bool       `json:"-"` // This value is used during parsing only

	Type             TaskType          `json:"type,omitempty" mapstructure:"type,omitempty"`
	Timer            string            `json:"timer,omitempty" mapstructure:"timer,omitempty"`
	Image            DockerImage       `json:"image"`
	VolumesFrom      []TaskName        `json:"volumes-from,omitempty"`
	Volumes          []string          `json:"volumes,omitempty"`
	Args             []string          `json:"args,omitempty"`
	Environment      map[string]string `json:"environment,omitempty"`
	Ports            []string          `json:"ports,omitempty"`
	PublicFrontEnds  []PublicFrontEnd  `json:"frontends,omitempty"`
	PrivateFrontEnds []PrivateFrontEnd `json:"private-frontends,omitempty"`
	HttpCheckPath    string            `json:"http-check-path,omitempty" mapstructure:"http-check-path,omitempty"`
	Capabilities     []string          `json:"capabilities,omitempty"`
	Links            []Link            `json:"links,omitempty"`
	Secrets          []Secret          `json:"secrets,omitempty"`
	Constraints      Constraints       `json:"constraints,omitempty"`
	DockerArgs       []string          `json:"docker-args,omitempty" mapstructure:"docker-args,omitempty"`
	LogDriver        LogDriver         `json:"log-driver,omitempty" mapstructure:"log-driver,omitempty"`
}

// Check for errors
func (t Task) Validate() error {
	if err := t.Name.Validate(); err != nil {
		return maskAny(err)
	}
	if err := t.Type.Validate(); err != nil {
		return maskAny(err)
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
	if err := t.Constraints.Validate(); err != nil {
		return maskAny(err)
	}
	if err := t.LogDriver.Validate(); err != nil {
		return maskAny(err)
	}
	return nil
}

// createUnits creates all units needed to run this task.
func (t *Task) createUnits(ctx generatorContext) ([]units.UnitChain, error) {
	mainChain := units.UnitChain{}

	proxyUnitNames := []string{}
	for _, l := range t.Links {
		if !l.Type.IsTCP() {
			continue
		}
		linkIndex := len(proxyUnitNames)
		unit, err := t.createProxyUnit(l, linkIndex, ctx)
		if err != nil {
			return nil, maskAny(err)
		}
		proxyUnitNames = append(proxyUnitNames, unit.FullName)
		mainChain = append(mainChain, unit)
	}

	main, err := t.createMainUnit(proxyUnitNames, ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	mainChain = append(mainChain, main)

	timer, err := t.createTimerUnit(ctx)
	if err != nil {
		return nil, maskAny(err)
	}

	chains := []units.UnitChain{mainChain}
	if timer != nil {
		timerChain := units.UnitChain{timer}
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
	base := strings.Replace(t.fullName(), "/", "-", -1)
	if t.group.Global {
		return base
	}
	return fmt.Sprintf("%s-%v", base, scalingGroup)
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
