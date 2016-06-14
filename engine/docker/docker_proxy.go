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

	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/pkg/cmdline"
)

// createProxyDockerCmdLine creates the `ExecStart` line for
// the proxy unit.
func (e *dockerEngine) createProxyDockerCmdLine(t *jobs.Task, containerName, containerImage string, link jobs.Link, env map[string]string, scalingGroup uint) (cmdline.Cmdline, error) {
	var cmd cmdline.Cmdline
	cmd.Add(nil, e.dockerPath, "run", "--rm", fmt.Sprintf("--name %s", containerName))
	for _, p := range link.Ports {
		cmd.Add(env, fmt.Sprintf("--expose %d", p))
	}
	cmd.Add(env, "-P")
	if e.options.EnvFile != "" {
		cmd.Add(env, fmt.Sprintf("--env-file=%s", e.options.EnvFile))
	}
	cmd.Add(env, "-e SERVICE_IGNORE=true") // Support registrator
	for _, arg := range t.LogDriver.CreateDockerLogArgs(e.options) {
		cmd.Add(env, arg)
	}

	cmd.Add(nil, containerImage)
	cmd.Add(env, fmt.Sprintf("--etcd-addr http://${COREOS_PRIVATE_IPV4}:2379/pulcy/service/%s", link.Target.EtcdServiceName()))

	return cmd, nil
}
