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
	"regexp"
	"strings"

	"github.com/juju/errgo"
)

const (
	defaultVolumePrefix = "/var/lib"
)

var (
	varRegexp = regexp.MustCompile(`\${[a-zA-Z0-9_\-\.]+}`)
)

type variableContext struct {
	Job   *Job
	Group *TaskGroup
	Task  *Task

	errors []string
}

func NewVariableContext(job *Job, group *TaskGroup, task *Task) *variableContext {
	return &variableContext{
		Job:   job,
		Group: group,
		Task:  task,
	}
}

func (ctx *variableContext) Err() error {
	if len(ctx.errors) > 0 {
		return maskAny(errgo.WithCausef(nil, ValidationError, strings.Join(ctx.errors, ", ")))
	}
	return nil
}

func (ctx *variableContext) replaceString(input string) string {
	assertJob := func(key string) bool {
		if ctx.Job != nil {
			return true
		}
		// No job given
		ctx.errors = append(ctx.errors, fmt.Sprintf("variable '%s' is not allowed outside a job", key))
		return false
	}

	assertGroup := func(key string) bool {
		if ctx.Group != nil {
			return true
		}
		// No job given
		ctx.errors = append(ctx.errors, fmt.Sprintf("variable '%s' is not allowed outside a group", key))
		return false
	}

	assertTask := func(key string) bool {
		if ctx.Task != nil {
			return true
		}
		// No job given
		ctx.errors = append(ctx.errors, fmt.Sprintf("variable '%s' is not allowed outside a task", key))
		return false
	}

	return varRegexp.ReplaceAllStringFunc(input, func(arg string) string {
		key := arg[2 : len(arg)-1]
		switch strings.TrimSpace(key) {
		case "job":
			if assertJob(key) {
				return ctx.Job.Name.String()
			}
		case "job.id":
			if assertJob(key) {
				return ctx.Job.ID
			}
		case "job.volume":
			if assertJob(key) {
				return fmt.Sprintf("%s/%s", defaultVolumePrefix, ctx.Job.Name)
			}
		case "group":
			if assertGroup(key) {
				return ctx.Group.Name.String()
			}
		case "group.full":
			if assertJob(key) && assertGroup(key) {
				return fmt.Sprintf("%s.%s", ctx.Job.Name, ctx.Group.Name)
			}
		case "group.volume":
			if assertJob(key) && assertGroup(key) {
				return fmt.Sprintf("%s/%s/%s", defaultVolumePrefix, ctx.Job.Name, ctx.Group.Name)
			}
		case "task":
			if assertTask(key) {
				return ctx.Task.Name.String()
			}
		case "task.full":
			if assertJob(key) && assertGroup(key) && assertTask(key) {
				return fmt.Sprintf("%s.%s.%s", ctx.Job.Name, ctx.Group.Name, ctx.Task.Name)
			}
		case "task.volume":
			if assertJob(key) && assertGroup(key) && assertTask(key) {
				return fmt.Sprintf("%s/%s/%s/%s", defaultVolumePrefix, ctx.Job.Name, ctx.Group.Name, ctx.Task.Name)
			}
		case "instance":
			if assertTask(key) {
				return "%i" // Will be expanded by Fleet/Systemd
			}
		case "instance.full":
			if assertJob(key) && assertGroup(key) && assertTask(key) {
				return fmt.Sprintf("%s.%s.%s@%%i", ctx.Job.Name, ctx.Group.Name, ctx.Task.Name)
			}
		case "container":
			if assertTask(key) {
				return ctx.Task.containerNameExt("%i")
			}
		}
		return arg
	})
}

func (ctx *variableContext) replaceStringSlice(input []string) []string {
	result := []string{}
	for _, x := range input {
		result = append(result, ctx.replaceString(x))
	}
	return result
}

func (ctx *variableContext) replaceStringMap(input map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range input {
		k = ctx.replaceString(k)
		v = ctx.replaceString(v)
		result[k] = v
	}
	return result
}
