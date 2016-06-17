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
	Secrets          SecretList        `json:"secrets,omitempty"`
	DockerArgs       []string          `json:"docker-args,omitempty" mapstructure:"docker-args,omitempty"`
	LogDriver        LogDriver         `json:"log-driver,omitempty" mapstructure:"log-driver,omitempty"`
	Target           LinkName          `json:"target,omitempty" mapstructure:"target,omitempty"`
	Rewrites         []Rewrite         `json:"rewrites,omitempty" mapstructure:"rewrites,omitempty"`
	User             string            `json:"user,omitempty" mapstructure:"user,omitempty"`
	Metrics          *Metrics          `json:"metrics,omitempty"`
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
	if t.Metrics != nil {
		m := t.Metrics.replaceVariables(ctx)
		t.Metrics = &m
	}
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
	if t.Metrics != nil {
		if err := t.Metrics.Validate(); err != nil {
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
	if err := t.Secrets.Validate(); err != nil {
		return maskAny(err)
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

// Task gets a task by the given name
func (t *Task) Task(name TaskName) (*Task, error) {
	result, err := t.group.Task(name)
	if err != nil {
		return nil, maskAny(err)
	}
	return result, nil
}

// JobID returns the ID of the job containing the group containing this task.
func (t *Task) JobID() string {
	return t.group.job.ID
}

// JobName returns the name of the job containing the group containing this task.
func (t *Task) JobName() JobName {
	return t.group.job.Name
}

// GroupGlobal returns true if the Global flag of the containing group is set.
func (t *Task) GroupGlobal() bool {
	return t.group.Global
}

// GroupCount returns the Count flag of the containing group.
func (t *Task) GroupCount() uint {
	return t.group.Count
}

// FullName returns the full name of this task: job/taskgroup/task
func (t *Task) FullName() string {
	return fmt.Sprintf("%s/%s", t.group.FullName(), t.Name)
}

// PrivateDomainName returns the DNS name (in the private namespace) for the given task.
func (t *Task) PrivateDomainName() string {
	ln := NewLinkName(t.group.job.Name, t.group.Name, t.Name, "")
	return ln.PrivateDomainName()
}

// InstanceSpecificPrivateDomainName returns the DNS name (in the private namespace) for an instance of the given task.
func (t *Task) InstanceSpecificPrivateDomainName(scalingGroup uint) string {
	ln := NewLinkName(t.group.job.Name, t.group.Name, t.Name, InstanceName(strconv.Itoa(int(scalingGroup))))
	return ln.PrivateDomainName()
}

// ServiceName returns the name used to register this service.
func (t *Task) ServiceName() string {
	return strings.Replace(t.FullName(), "/", "-", -1)
}

// ContainerName returns the name of the docker container used for this task.
func (t *Task) ContainerName(scalingGroup uint) string {
	return t.containerNameExt(strconv.Itoa(int(scalingGroup)))
}

// containerName returns the name of the docker contained used for this task.
func (t *Task) containerNameExt(instance string) string {
	base := strings.Replace(t.FullName(), "/", "-", -1)
	if t.group.Global {
		return base
	}
	return fmt.Sprintf("%s-%s", base, instance)
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

// MergedConstraints returns the constraints resulting from merging the job constraints
// with the group constraints.
func (t *Task) MergedConstraints() Constraints {
	return t.group.job.Constraints.Merge(t.group.Constraints)
}
