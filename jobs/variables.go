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
	"strconv"
	"strings"

	"github.com/juju/errgo"
)

const (
	defaultVolumePrefix = "/var/lib"
)

var (
	varRegexp = regexp.MustCompile(`\${[a-zA-Z0-9_\-\.\ @]+}`)
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

func (ctx *variableContext) assertJob(key string) bool {
	if ctx.Job != nil {
		return true
	}
	// No job given
	ctx.errors = append(ctx.errors, fmt.Sprintf("variable '%s' is not allowed outside a job", key))
	return false
}

func (ctx *variableContext) assertGroup(key string) bool {
	if ctx.Group != nil {
		return true
	}
	// No job given
	ctx.errors = append(ctx.errors, fmt.Sprintf("variable '%s' is not allowed outside a group", key))
	return false
}

func (ctx *variableContext) assertTask(key string) bool {
	if ctx.Task != nil {
		return true
	}
	// No job given
	ctx.errors = append(ctx.errors, fmt.Sprintf("variable '%s' is not allowed outside a task", key))
	return false
}

func (ctx *variableContext) replaceString(input string) string {
	return varRegexp.ReplaceAllStringFunc(input, func(arg string) string {
		key := arg[2 : len(arg)-1]
		switch strings.TrimSpace(key) {
		case "job":
			if ctx.assertJob(key) {
				return ctx.Job.Name.String()
			}
		case "job.id":
			if ctx.assertJob(key) {
				return ctx.Job.ID
			}
		case "job.volume":
			if ctx.assertJob(key) {
				return fmt.Sprintf("%s/%s", defaultVolumePrefix, ctx.Job.Name)
			}
		case "group":
			if ctx.assertGroup(key) {
				return ctx.Group.Name.String()
			}
		case "group.full":
			if ctx.assertJob(key) && ctx.assertGroup(key) {
				return fmt.Sprintf("%s.%s", ctx.Job.Name, ctx.Group.Name)
			}
		case "group.volume":
			if ctx.assertJob(key) && ctx.assertGroup(key) {
				return fmt.Sprintf("%s/%s/%s", defaultVolumePrefix, ctx.Job.Name, ctx.Group.Name)
			}
		case "task":
			if ctx.assertTask(key) {
				return ctx.Task.Name.String()
			}
		case "task.full":
			if ctx.assertJob(key) && ctx.assertGroup(key) && ctx.assertTask(key) {
				return fmt.Sprintf("%s.%s.%s", ctx.Job.Name, ctx.Group.Name, ctx.Task.Name)
			}
		case "task.volume":
			if ctx.assertJob(key) && ctx.assertGroup(key) && ctx.assertTask(key) {
				return fmt.Sprintf("%s/%s/%s/%s", defaultVolumePrefix, ctx.Job.Name, ctx.Group.Name, ctx.Task.Name)
			}
		case "instance":
			if ctx.assertTask(key) {
				// TODO this is renderer specific
				return "%i" // Will be expanded by Fleet/Systemd
			}
		case "instance.full":
			if ctx.assertJob(key) && ctx.assertGroup(key) && ctx.assertTask(key) {
				return fmt.Sprintf("%s.%s.%s@%%i", ctx.Job.Name, ctx.Group.Name, ctx.Task.Name)
			}
		case "container":
			if ctx.assertTask(key) {
				return ctx.Task.containerNameExt("%i") // TODO this is renderer specific
			}
		case "private_ipv4":
			return "${COREOS_PRIVATE_IPV4}"
		case "public_ipv4":
			return "${COREOS_PUBLIC_IPV4}"
		case "etcd_endpoints":
			return "${ETCD_ENDPOINTS}"
		case "etcd_host":
			return "${ETCD_HOST}"
		case "etcd_port":
			return "${ETCD_PORT}"
		case "hostname":
			return "%H" // TODO this is renderer specific
		case "machine_id":
			return "%m" // TODO this is renderer specific
		default:
			parts := strings.Split(key, " ")
			assertNoArgs := func(noArgs int) bool {
				if (len(parts) - 1) == noArgs {
					return true
				}
				ctx.errors = append(ctx.errors, fmt.Sprintf("variable '%s' expects %d arguments, got %d", parts[0], noArgs, len(parts)-1))
				return false
			}
			switch parts[0] {
			case "link_tcp":
				if ctx.assertTask(key) && assertNoArgs(2) {
					target := ctx.findTarget(key, parts[1])
					port, err := strconv.Atoi(parts[2])
					if err != nil {
						ctx.errors = append(ctx.errors, fmt.Sprintf("variable '%s' expects a port argument, got '%s'", parts[0], parts[2]))
					} else {
						if ctx.Task.Network.IsWeave() && !target.HasInstance() {
							targetTask, err := ctx.findTargetTask(key, parts[1])
							if err != nil {
								ctx.errors = append(ctx.errors, fmt.Sprintf("unknown target '%s'", parts[1]))
							}
							if targetTask.Type.IsService() && targetTask.Network.IsWeave() {
								return fmt.Sprintf("tcp://%s:%d", targetTask.WeaveDomainName(), port)
							}
						}
						ctx.Task.Links = ctx.Task.Links.Add(Link{
							Type:   LinkTypeTCP,
							Target: target,
							Ports:  []int{port},
						})
						url, _ := linkTCP(string(target), port)
						return url
					}
				}
			case "link_url":
				if ctx.assertTask(key) && assertNoArgs(1) {
					target := ctx.findTarget(key, parts[1])
					if ctx.Task.Network.IsWeave() && !target.HasInstance() {
						targetTask, err := ctx.findTargetTask(key, parts[1])
						if err != nil {
							ctx.errors = append(ctx.errors, fmt.Sprintf("unknown target '%s'", parts[1]))
							return ""
						}
						if targetTask.Type.IsService() && targetTask.Network.IsWeave() {
							// We can use a direct weave DNS link
							targetPort := targetTask.PrivateFrontEndPort(80)
							if targetPort != 80 {
								return fmt.Sprintf("http://%s:%d", targetTask.WeaveDomainName(), targetPort)
							} else {
								return fmt.Sprintf("http://%s", targetTask.WeaveDomainName())
							}
						}
						if targetTask.Type.IsProxy() {
							proxyTask := targetTask
							proxyTarget := ctx.Task.resolveLink(proxyTask.Target) // Target is not linked yet, so do that here.
							if len(proxyTask.PrivateFrontEnds) == 1 {
								proxyPort := proxyTask.PrivateFrontEndPort(80)
								if !proxyTarget.HasInstance() {
									targetTask, err = ctx.findTask(proxyTarget)
									if err != nil {
										ctx.errors = append(ctx.errors, fmt.Sprintf("unknown target '%s' of proxy '%s'", proxyTarget, proxyTask.Name))
										return ""
									}
									if targetTask.Network.IsWeave() {
										r := proxyTask.Rewrite
										if r == nil {
											// We can use a direct weave DNS link
											return fmt.Sprintf("http://%s:%d", targetTask.WeaveDomainName(), proxyPort)
										} else if r.HasPathPrefixOnly() {
											// We can use a direct weave DNS link with a path prefix
											path := strings.TrimSuffix(strings.TrimPrefix(r.PathPrefix, "/"), "/")
											return fmt.Sprintf("http://%s:%d/%s", targetTask.WeaveDomainName(), proxyPort, path)
										}
									}
								}
							}
						}
					}
					ctx.Task.Links = ctx.Task.Links.Add(Link{
						Target: target,
					})
					url, _ := linkURL(string(target))
					return url
				}
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

func (ctx *variableContext) findTarget(key, name string) LinkName {
	ln := LinkName(name)
	j, tg, t, i, _ := ln.parse()
	if j == "" && ctx.assertGroup(key) {
		j = ctx.Job.Name
	}
	if tg == "" && ctx.assertGroup(key) {
		tg = ctx.Group.Name
	}
	if t == "" && ctx.assertTask(key) {
		t = ctx.Task.Name
	}
	return NewLinkName(j, tg, t, i)
}

func (ctx *variableContext) findTask(ln LinkName) (*Task, error) {
	jn, _ := ln.Job()
	if jn != ctx.Job.Name {
		return nil, maskAny(errgo.WithCausef(nil, TaskNotFoundError, "Job '%s' not found", jn))
	}
	tgn, _ := ln.TaskGroup()
	tg, err := ctx.Job.TaskGroup(tgn)
	if err != nil {
		return nil, maskAny(err)
	}
	tn, _ := ln.Task()
	t, err := tg.Task(tn)
	if err != nil {
		return nil, maskAny(err)
	}
	in, _ := ln.Instance()
	if !in.IsEmpty() {
		return nil, maskAny(errgo.WithCausef(nil, TaskNotFoundError, "Instance of '%s' should be empty", ln))
	}
	return t, nil
}

func (ctx *variableContext) findTargetTask(key, name string) (*Task, error) {
	ln := ctx.findTarget(key, name)
	return ctx.findTask(ln)
}
