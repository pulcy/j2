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
	renderer Renderer
	Job      *Job
	Group    *TaskGroup
	Task     *Task

	errors []string
}

func NewVariableContext(renderer Renderer, job *Job, group *TaskGroup, task *Task) *variableContext {
	return &variableContext{
		renderer: renderer,
		Job:      job,
		Group:    group,
		Task:     task,
	}
}

func (ctx *variableContext) Err() error {
	if len(ctx.errors) > 0 {
		return maskAny(errgo.WithCausef(nil, ValidationError, strings.Join(ctx.errors, ", ")))
	}
	return nil
}

func (ctx *variableContext) String() string {
	r := "ctx "
	if ctx.Job != nil {
		r = fmt.Sprintf("%s job=%s", ctx.Job.Name)
		if ctx.Group != nil {
			r = fmt.Sprintf("%s group=%s", ctx.Group.Name)
			if ctx.Task != nil {
				r = fmt.Sprintf("%s task=%s", ctx.Task.Name)
			}
		}
	}
	return r
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
	r := ctx.renderer
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
				return r.ExpandInstance()
			}
		case "instance.full":
			if ctx.assertJob(key) && ctx.assertGroup(key) && ctx.assertTask(key) {
				return fmt.Sprintf("%s.%s.%s@%s", ctx.Job.Name, ctx.Group.Name, ctx.Task.Name, r.ExpandInstance())
			}
		case "container":
			if ctx.assertTask(key) {
				return ctx.Task.containerNameExt(r.ExpandInstance())
			}
		case "private_ipv4":
			return r.ExpandPrivateIPv4()
		case "public_ipv4":
			return r.ExpandPublicIPv4()
		case "etcd_endpoints":
			return r.ExpandEtcdEndpoints()
		case "etcd_host":
			return r.ExpandEtcdHost()
		case "etcd_port":
			return r.ExpandEtcdPort()
		case "hostname":
			return r.ExpandHostname()
		case "machine_id":
			return r.ExpandMachineID()
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
						if r.SupportsDNSLinkTo(ctx.Task, target) {
							targetTask, err := ctx.findTargetTask(key, parts[1])
							if err != nil {
								ctx.errors = append(ctx.errors, fmt.Sprintf("link_tcp: unknown target '%s' in '%s'", parts[1], ctx))
							}
							if r.TaskAcceptsDNSLink(targetTask) {
								return createURL("tcp", r.TaskDNSName(targetTask), port, -1, "")
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
					if r.SupportsDNSLinkTo(ctx.Task, target) {
						targetName := ctx.findTarget(key, parts[1])
						if !ctx.isSameJob(targetName) {
							dependency, err := ctx.Job.Dependency(targetName)
							if err != nil {
								ctx.errors = append(ctx.errors, fmt.Sprintf("link_url: unknown external target '%s' in '%s'", targetName, ctx))
								return ""
							}
							if r.DependencyAcceptsDNSLink(dependency) {
								targetPort := dependency.PrivateFrontEndPort(80)
								return createURL("http", r.DependencyDNSName(dependency), targetPort, 80, "")
							}
						}
						targetTask, err := ctx.findTask(targetName)
						if err != nil {
							ctx.errors = append(ctx.errors, fmt.Sprintf("link_url: unknown target '%s' in '%s'", parts[1], ctx))
							return ""
						}
						if r.TaskAcceptsDNSLink(targetTask) {
							// We can use a direct weave DNS link
							targetPort := targetTask.PrivateFrontEndPort(80)
							return createURL("http", r.TaskDNSName(targetTask), targetPort, 80, "")
						}
						if targetTask.Type.IsProxy() {
							proxyTask := targetTask
							proxyTarget := ctx.Task.resolveLink(proxyTask.Target) // Target is not linked yet, so do that here.
							if len(proxyTask.PrivateFrontEnds) == 1 {
								proxyPort := proxyTask.PrivateFrontEndPort(80)
								if !proxyTarget.HasInstance() {
									var targetAcceptsDNS bool
									var targetDomainName string
									if ctx.isSameJob(proxyTarget) {
										targetTask, err := ctx.findTask(proxyTarget)
										if err != nil {
											ctx.errors = append(ctx.errors, fmt.Sprintf("link_url (proxy): unknown target '%s' of proxy '%s'", proxyTarget, proxyTask.Name))
											return ""
										}
										targetAcceptsDNS = r.TaskAcceptsDNSLink(targetTask)
										targetDomainName = r.TaskDNSName(targetTask)
									} else {
										dependency, err := ctx.Job.Dependency(proxyTarget)
										if err != nil {
											ctx.errors = append(ctx.errors, fmt.Sprintf("link_url (proxy): unknown external target '%s' of proxy '%s'", proxyTarget, proxyTask.Name))
											return ""
										}
										targetAcceptsDNS = r.DependencyAcceptsDNSLink(dependency)
										targetDomainName = r.DependencyDNSName(dependency)
									}
									if targetAcceptsDNS {
										r := proxyTask.Rewrite
										if r == nil {
											// We can use a direct DNS link
											return createURL("http", targetDomainName, proxyPort, 80, "")
										} else if r.HasPathPrefixOnly() {
											// We can use a direct DNS link with a path prefix
											path := strings.TrimSuffix(strings.TrimPrefix(r.PathPrefix, "/"), "/")
											return createURL("http", targetDomainName, proxyPort, 80, path)
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

func (ctx *variableContext) isSameJob(name LinkName) bool {
	jn, _ := name.Job()
	return ctx.Job != nil && ctx.Job.Name == jn
}

func createURL(scheme, host string, port, defaultPort int, relPath string) string {
	relPath = strings.TrimSuffix(relPath, "/")
	if relPath != "" {
		relPath = "/" + relPath
	}
	if port == defaultPort {
		return fmt.Sprintf("%s://%s%s", scheme, host, relPath)
	}
	return fmt.Sprintf("%s://%s:%d%s", scheme, host, port, relPath)
}
