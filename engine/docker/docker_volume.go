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

// CreateVolumeCmds creates the commands needed for a volume unit.
func (e *dockerEngine) CreateVolumeCmds(t *jobs.Task, vol jobs.Volume, volIndex int, volPrefix, volHostPath string, env map[string]string, scalingGroup uint) (engine.Cmds, error) {
	containerImage := e.images.CephVolume
	containerName := createVolumeUnitContainerName(t, volIndex, scalingGroup)
	execStart, err := e.createVolumeDockerCmdLine(t, containerName, containerImage, vol, volPrefix, volHostPath, env, scalingGroup)
	if err != nil {
		return engine.Cmds{}, maskAny(err)
	}
	testVolHostPathCmd := e.createTestLocalVolumeCmd(volHostPath)

	var cmds engine.Cmds
	cmds.Start = append(cmds.Start,
		e.pullCmd(containerImage),
		testVolHostPathCmd,
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

// createVolumeDockerCmdLine creates the `ExecStart` line for
// the volume unit.
func (e *dockerEngine) createVolumeDockerCmdLine(t *jobs.Task, containerName, containerImage string, vol jobs.Volume, volPrefix, volHostPath string, env map[string]string, scalingGroup uint) (cmdline.Cmdline, error) {
	var cmd cmdline.Cmdline
	cmd.Add(nil, e.dockerPath, "run", "--rm", fmt.Sprintf("--name %s", containerName), "--net=host", "--privileged")

	cmd.Add(env, fmt.Sprintf("-v %s:%s:shared", volHostPath, vol.Path))
	cmd.Add(env, "-v /usr/bin/etcdctl:/usr/bin/etcdctl")
	if e.options.EnvFile != "" {
		cmd.Add(env, fmt.Sprintf("--env-file=%s", e.options.EnvFile))
	}
	cmd.Add(env, "-e SERVICE_IGNORE=true") // Support registrator
	cmd.Add(env, "-e PREFIX="+volPrefix)
	cmd.Add(env, "-e TARGET="+vol.Path)
	cmd.Add(env, "-e WAIT=1")
	if v, err := vol.MountOption("uid"); err == nil {
		cmd.Add(env, "-e UID="+v)
	}
	if v, err := vol.MountOption("gid"); err == nil {
		cmd.Add(env, "-e GID="+v)
	}
	for _, arg := range t.LogDriver.CreateDockerLogArgs(e.options) {
		cmd.Add(env, arg)
	}

	cmd.Add(nil, containerImage)

	return cmd, nil
}

func (e *dockerEngine) createTestLocalVolumeCmd(volHostPath string) cmdline.Cmdline {
	return *cmdline.New(nil, e.shPath, "-c", fmt.Sprintf("'test -e %s || mkdir -p %s'", volHostPath, volHostPath))
}
