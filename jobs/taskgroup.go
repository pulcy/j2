package jobs

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"

	"github.com/juju/errgo"

	"arvika.pulcy.com/pulcy/deployit/units"
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

	Count  uint     `json:"count"`            // Number of instances of this group
	Global bool     `json:"global,omitempty"` // Scheduled on all machines
	Tasks  TaskList `json:"tasks"`
}

type TaskGroupList []*TaskGroup

// Link objects just after parsing
func (tg *TaskGroup) link() {
	for _, v := range tg.Tasks {
		v.group = tg
	}
	sort.Sort(tg.Tasks)
}

// Check for configuration errors
func (tg *TaskGroup) Validate() error {
	if err := tg.Name.Validate(); err != nil {
		return maskAny(err)
	}
	if tg.Count == 0 {
		return maskAny(errgo.WithCausef(nil, ValidationError, "group %s count 0", tg.Name))
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
func (tg *TaskGroup) IsScalable() bool {
	return !tg.Global
}

// createUnits creates all units needed to run this taskgroup.
func (tg *TaskGroup) createUnits(ctx generatorContext) ([]*units.Unit, error) {
	if tg.Global {
		if ctx.ScalingGroup != 1 {
			return nil, nil
		}
	} else {
		if ctx.ScalingGroup > tg.Count {
			return nil, nil
		}
	}

	// Create all units for my tasks
	units := []*units.Unit{}
	for _, t := range tg.Tasks {
		taskUnits, err := t.createUnits(ctx)
		if err != nil {
			return nil, maskAny(err)
		}
		units = append(units, taskUnits...)
	}

	// Force units to be on the same machine
	if !tg.Global {
		groupUnitsOnMachine(units)
	}

	return units, nil
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
