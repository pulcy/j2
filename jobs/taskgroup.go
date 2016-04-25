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
	"bytes"
	"fmt"
	"regexp"
	"sort"

	"github.com/juju/errgo"

	"github.com/pulcy/j2/cluster"
	"github.com/pulcy/j2/units"
)

const (
	defaultCount = uint(1)
)

var (
	taskGroupNamePattern = regexp.MustCompile(`^([a-z0-9_]{2,30})$`)
)

type TaskGroupName string

func (tgn TaskGroupName) String() string {
	return string(tgn)
}

func (tgn TaskGroupName) Validate() error {
	if !taskGroupNamePattern.MatchString(string(tgn)) {
		return maskAny(errgo.WithCausef(nil, InvalidNameError, "taskgroup name must match '%s', got '%s'", taskGroupNamePattern, tgn))
	}
	return nil
}

// TaskGroup is a group of tasks that are scheduled on the same
// machine.
// TaskGroups can have multiple instances, specified by `Count`.
// Multiple instances are scheduled on different machines when possible.
type TaskGroup struct {
	Name TaskGroupName `json:"name", mapstructure:"-"`
	job  *Job

	Count         uint          `json:"count"`            // Number of instances of this group
	Global        bool          `json:"global,omitempty"` // Scheduled on all machines
	Tasks         TaskList      `json:"tasks"`
	Constraints   Constraints   `json:"constraints,omitempty"`
	RestartPolicy RestartPolicy `json:"restart,omitempty" mapstructure:"restart,omitempty"`
}

type TaskGroupList []*TaskGroup

// Link objects just after parsing
func (tg *TaskGroup) prelink() {
	for _, v := range tg.Tasks {
		v.group = tg
	}
}

// Link objects just after replacing variables
func (tg *TaskGroup) link() {
	for _, v := range tg.Tasks {
		v.link()
	}
	sort.Sort(tg.Tasks)
	sort.Sort(tg.Constraints)
}

// optimizeFor optimizes the task group for the given cluster.
func (tg *TaskGroup) optimizeFor(cluster cluster.Cluster) {
	for _, t := range tg.Tasks {
		t.optimizeFor(cluster)
	}
	if tg.Global && int(tg.Count) > cluster.InstanceCount {
		tg.Count = uint(cluster.InstanceCount)
	}
}

// replaceVariables replaces all known variables in the values of the given group.
func (tg *TaskGroup) replaceVariables() error {
	ctx := NewVariableContext(tg.job, tg, nil)
	for _, x := range tg.Tasks {
		if err := x.replaceVariables(); err != nil {
			return maskAny(err)
		}
	}
	for i, x := range tg.Constraints {
		tg.Constraints[i] = x.replaceVariables(ctx)
	}
	tg.RestartPolicy = RestartPolicy(ctx.replaceString(string(tg.RestartPolicy)))
	return maskAny(ctx.Err())
}

// Check for configuration errors
func (tg *TaskGroup) Validate() error {
	if err := tg.Name.Validate(); err != nil {
		return maskAny(err)
	}
	if tg.Count <= 0 {
		return maskAny(errgo.WithCausef(nil, ValidationError, "group %s count <= 0", tg.Name))
	}
	if len(tg.Tasks) == 0 {
		return maskAny(errgo.WithCausef(nil, ValidationError, "group %s has no tasks", tg.Name))
	}
	for i, t := range tg.Tasks {
		err := t.Validate()
		if err != nil {
			return maskAny(err)
		}
		for j := i + 1; j < len(tg.Tasks); j++ {
			if tg.Tasks[j].Name == t.Name {
				return maskAny(errgo.WithCausef(nil, ValidationError, "group %s has duplicate task %s", tg.Name, t.Name))
			}
		}
	}
	if err := tg.Constraints.Validate(); err != nil {
		return maskAny(err)
	}
	if err := tg.RestartPolicy.Validate(); err != nil {
		return maskAny(err)
	}
	return nil
}

// Task gets a task by the given name
func (tg *TaskGroup) Task(name TaskName) (*Task, error) {
	for _, t := range tg.Tasks {
		if t.Name == name {
			return t, nil
		}
	}
	return nil, maskAny(errgo.WithCausef(nil, TaskNotFoundError, name.String()))
}

// Is this group scalable?
// That mean "not global"
/*func (tg *TaskGroup) IsScalable() bool {
	return !tg.Global
}*/

type taskUnitChain struct {
	Task      *Task
	MainChain units.UnitChain
}

type taskUnitChainList []taskUnitChain

func (l taskUnitChainList) find(taskName TaskName) (taskUnitChain, error) {
	for _, x := range l {
		if x.Task.Name == taskName {
			return x, nil
		}
	}
	return taskUnitChain{}, maskAny(errgo.WithCausef(nil, TaskNotFoundError, taskName.String()))
}

// createUnits creates all units needed to run this taskgroup.
func (tg *TaskGroup) createUnits(ctx generatorContext) ([]units.UnitChain, error) {
	if ctx.ScalingGroup > tg.Count {
		return nil, nil
	}

	// Create all units for my tasks
	allChains := []units.UnitChain{}
	taskUnitChains := taskUnitChainList{}
	allUnits := []*units.Unit{}
	for _, t := range tg.Tasks {
		tuc, err := t.createUnits(ctx)
		if err != nil {
			return nil, maskAny(err)
		}
		allChains = append(allChains, tuc...)
		if len(tuc) > 0 {
			taskUnitChains = append(taskUnitChains, taskUnitChain{
				Task:      t,
				MainChain: tuc[0],
			})
		}
		// Link chains to enfore the actual chain
		for _, chain := range tuc {
			chain.Link()
		}
		// Collect all units in the chain
		for _, chain := range tuc {
			allUnits = append(allUnits, chain...)
		}
	}

	// In case of restart="all", bind chains such that they restart together
	if tg.RestartPolicy.IsAll() {
		for i, x := range taskUnitChains {
			for j, y := range taskUnitChains {
				if i == j {
					continue
				}
				x.MainChain.BindRestartTo(y.MainChain)
			}
		}
	}

	// Create "After" links
	for _, x := range taskUnitChains {
		for _, afterName := range x.Task.After {
			other, err := taskUnitChains.find(afterName)
			if err != nil {
				return nil, maskAny(err)
			}
			x.MainChain.After(other.MainChain)
		}
	}

	// Force units to be on the same machine
	if !tg.Global {
		units.GroupUnitsOnMachine(allUnits...)
	}

	return allChains, nil
}

// Gets the full name of this taskgroup: job/taskgroup
func (tg *TaskGroup) fullName() string {
	return fmt.Sprintf("%s/%s", tg.job.Name, tg.Name)
}

func (l TaskGroupList) Len() int {
	return len(l)
}

func (l TaskGroupList) Less(i, j int) bool {
	return bytes.Compare([]byte(l[i].Name.String()), []byte(l[j].Name.String())) < 0
}

func (l TaskGroupList) Swap(i, j int) {
	tmp := l[i]
	l[i] = l[j]
	l[j] = tmp
}
