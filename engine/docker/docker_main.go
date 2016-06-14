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

package docker

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/pkg/cmdline"
)

// createMainDockerCmdLine creates the `ExecStart` line for
// the main unit.
func (e *dockerEngine) createMainDockerCmdLine(t *jobs.Task, image string, env map[string]string, scalingGroup uint) (cmdline.Cmdline, error) {
	serviceName := t.ServiceName()
	var cmd cmdline.Cmdline
	cmd.Add(nil, e.dockerPath, "run", "--rm", fmt.Sprintf("--name %s", t.ContainerName(scalingGroup)))
	if len(t.Ports) > 0 {
		for _, p := range t.Ports {
			cmd.Add(env, fmt.Sprintf("-p %s", p))
		}
	} else {
		cmd.Add(env, "-P")
	}
	for i, v := range t.Volumes {
		if v.IsLocal() {
			cmd.Add(env, fmt.Sprintf("-v %s", v))
		} else if !v.IsLocal() {
			cmd.Add(env, fmt.Sprintf("--volumes-from %s", createVolumeUnitContainerName(t, i, scalingGroup)))
		}
	}
	for _, secret := range t.Secrets {
		if ok, path := secret.TargetFile(); ok {
			hostPath, err := secretFilePath(t, scalingGroup, secret)
			if err != nil {
				return cmdline.Cmdline{}, maskAny(err)
			}
			cmd.Add(env, fmt.Sprintf("-v %s:%s:ro", hostPath, path))
		}
	}
	for _, name := range t.VolumesFrom {
		other, err := t.Task(name)
		if err != nil {
			return cmdline.Cmdline{}, maskAny(err)
		}
		for i, v := range other.Volumes {
			if !v.IsLocal() {
				cmd.Add(env, fmt.Sprintf("--volumes-from %s", createVolumeUnitContainerName(other, i, scalingGroup)))
			}
		}
		cmd.Add(env, fmt.Sprintf("--volumes-from %s", other.ContainerName(scalingGroup)))
	}
	envKeys := []string{}
	for k := range t.Environment {
		envKeys = append(envKeys, k)
	}
	sort.Strings(envKeys)
	if e.options.EnvFile != "" {
		cmd.Add(env, fmt.Sprintf("--env-file=%s", e.options.EnvFile))
	}
	for _, k := range envKeys {
		cmd.Add(env, "-e "+strconv.Quote(fmt.Sprintf("%s=%s", k, t.Environment[k])))
	}
	if hasEnvironmentSecrets(t) {
		cmd.Add(env, "--env-file="+secretEnvironmentPath(t, scalingGroup))
	}
	cmd.Add(env, fmt.Sprintf("-e SERVICE_NAME=%s", serviceName)) // Support registrator
	for _, cap := range t.Capabilities {
		cmd.Add(env, "--cap-add "+cap)
	}
	tcpLinkIndex := 0
	for _, l := range t.Links {
		targetName := l.Target.PrivateDomainName()
		if l.Type.IsHTTP() {
			cmd.Add(env, "--add-host")
			cmd.Add(env, fmt.Sprintf("%s:${COREOS_PRIVATE_IPV4}", targetName))
		} else {
			linkContainerName := fmt.Sprintf("%s-pr%d", t.ContainerName(scalingGroup), tcpLinkIndex)
			cmd.Add(env, fmt.Sprintf("--link %s:%s", linkContainerName, targetName))
			tcpLinkIndex++
		}
	}
	for _, arg := range t.LogDriver.CreateDockerLogArgs(e.options) {
		cmd.Add(env, arg)
	}
	for _, arg := range t.DockerArgs {
		cmd.Add(env, arg)
	}
	if t.User != "" {
		cmd.Add(env, fmt.Sprintf("--user %s", t.User))
	}

	cmd.Add(nil, image)
	if t.Type == "proxy" {
		cmd.Add(nil, "sleep 36500d")
	}
	for _, arg := range t.Args {
		cmd.Add(env, arg)
	}

	return cmd, nil
}

// hasEnvironmentSecrets returns true if the given task has secrets that should
// be stored in an environment variable. False otherwise.
func hasEnvironmentSecrets(t *jobs.Task) bool {
	for _, secret := range t.Secrets {
		if ok, _ := secret.TargetEnviroment(); ok {
			return true
		}
	}
	return false
}
