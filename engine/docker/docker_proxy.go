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

	"github.com/pulcy/j2/engine"
	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/pkg/cmdline"
)

// CreateProxyCmds creates the commands needed for a proxy unit.
func (e *dockerEngine) CreateProxyCmds(t *jobs.Task, link jobs.Link, linkIndex int, env map[string]string, scalingGroup uint) (engine.Cmds, error) {
	containerName := fmt.Sprintf("%s-pr%d", t.ContainerName(scalingGroup), linkIndex)
	containerImage := images.Wormhole
	execStart, err := e.createProxyDockerCmdLine(t, containerName, containerImage, link, env, scalingGroup)
	if err != nil {
		return engine.Cmds{}, maskAny(err)
	}
	var cmds engine.Cmds
	cmds.Start = append(cmds.Start,
		e.pullCmd(containerImage),
		e.stopCmd(containerName),
		e.removeCmd(containerName),
		e.cleanupCmd(),
	)
	if e.options.EnvFile != "" {
		cmds.Start = append(cmds.Start, *cmdline.New(nil, e.touchPath, e.options.EnvFile))
	}
	cmds.Start = append(cmds.Start, execStart)

	cmds.Stop = append(cmds.Stop,
		e.stopCmd(containerName),
		e.removeCmd(containerName),
	)

	return cmds, nil
}

// createProxyDockerCmdLine creates the `ExecStart` line for
// the proxy unit.
func (e *dockerEngine) createProxyDockerCmdLine(t *jobs.Task, containerName, containerImage string, link jobs.Link, env map[string]string, scalingGroup uint) (cmdline.Cmdline, error) {
	var cmd cmdline.Cmdline
	cmd, err := e.createDockerCmd(env, t.Network)
	if err != nil {
		return cmd, maskAny(err)
	}
	cmd.Add(nil, "run", "--rm", fmt.Sprintf("--name %s", containerName))
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
	cmd.Add(env, "--etcd-endpoint=${ETCD_ENDPOINTS}")
	cmd.Add(env, fmt.Sprintf("--etcd-path=/pulcy/service/%s", link.Target.EtcdServiceName()))

	return cmd, nil
}
