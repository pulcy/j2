package jobs

import (
	"fmt"
	"regexp"

	"github.com/juju/errgo"
)

var (
	taskGroupNamePattern = regexp.MustCompile(`^([a-z0-9_]{3,30})$`)
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
	Name TaskGroupName `json:"-"`
	Job  *Job          `json:"-"`

	Count int                `json:"count"` // Number of instances of this group
	Tasks map[TaskName]*Task `json:"tasks"`
}

// Link objects just after parsing
func (tg *TaskGroup) link() {
	for k, v := range tg.Tasks {
		v.Name = k
		v.Group = tg
	}
}

// Check for configuration errors
func (tg *TaskGroup) Validate() error {
	if err := tg.Name.Validate(); err != nil {
		return maskAny(err)
	}
	for _, t := range tg.Tasks {
		err := t.Validate()
		if err != nil {
			return maskAny(err)
		}
	}
	return nil
}

// Task gets a task by the given name
func (tg *TaskGroup) Task(name TaskName) (*Task, error) {
	if t, ok := tg.Tasks[name]; ok {
		return t, nil
	} else {
		return nil, maskAny(errgo.WithCausef(nil, TaskNotFoundError, name.String()))
	}
}

// Gets the full name of this taskgroup: job/taskgroup
func (tg *TaskGroup) fullName() string {
	return fmt.Sprintf("%s/%s", tg.Job.Name, tg.Name)
}
