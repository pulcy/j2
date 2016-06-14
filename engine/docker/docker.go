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

	"github.com/pulcy/j2/cluster"
	"github.com/pulcy/j2/engine"
	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/pkg/cmdline"
)

type Images struct {
	VaultMonkey string // Docker image name of vault-monkey
	Wormhole    string // Docker image name of wormhole
	Alpine      string // Docker image name of alpine linux
	CephVolume  string // Docker image name of ceph-volume
}

var (
	images Images
)

// SetupImages performs a global setup of the compiled in docker image names+versions.
func SetupImages(i Images) {
	images = i
}

type dockerEngine struct {
	options                 cluster.DockerOptions
	images                  Images
	dockerPath              string
	cleanupScriptPath       string
	containerTimeoutStopSec int
}

// NewDockerEngine creates a new engine renderer for docker
func NewDockerEngine(options cluster.DockerOptions) engine.Engine {
	return &dockerEngine{
		options:                 options,
		dockerPath:              "/usr/bin/docker",
		cleanupScriptPath:       "/home/core/bin/docker-cleanup.sh",
		containerTimeoutStopSec: 10,
	}
}

func (e *dockerEngine) CreateMainCmds(t *jobs.Task, env map[string]string, scalingGroup uint) (engine.Cmds, error) {
	containerName := t.ContainerName(scalingGroup)
	containerImage := t.Image.String()
	if t.Type == "proxy" {
		containerImage = images.Alpine
	}
	execStart, err := e.createMainDockerCmdLine(t, containerImage, env, scalingGroup)
	if err != nil {
		return engine.Cmds{}, maskAny(err)
	}

	var cmds engine.Cmds
	cmds.Start = append(cmds.Start,
		e.pullCmd(containerImage),
	)
	if e.options.EnvFile != "" {
		cmds.Start = append(cmds.Start, *cmdline.New(nil, "/usr/bin/touch", e.options.EnvFile))
	}
	// Add secret extraction commands
	secretsCmds, err := e.createSecretsExecStartPre(t, images.VaultMonkey, env, scalingGroup)
	if err != nil {
		return engine.Cmds{}, maskAny(err)
	}
	cmds.Start = append(cmds.Start, secretsCmds...)
	cmds.Start = append(cmds.Start,
		e.stopCmd(containerName),
		e.removeCmd(containerName),
		e.cleanupCmd(),
	)

	for _, v := range t.Volumes {
		if v.IsLocal() {
			hostPath := v.HostPath
			cmds.Start = append(cmds.Start, *cmdline.New(nil, "/bin/sh", "-c", fmt.Sprintf("'test -e %s || mkdir -p %s'", hostPath, hostPath)))
		}
	}

	cmds.Start = append(cmds.Start, execStart)

	cmds.Stop = append(cmds.Stop,
		e.stopCmd(containerName),
		e.removeCmd(containerName),
	)

	return cmds, nil

}

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
		cmds.Start = append(cmds.Start, *cmdline.New(nil, "/usr/bin/touch", e.options.EnvFile))
	}
	cmds.Start = append(cmds.Start, execStart)

	cmds.Stop = append(cmds.Stop,
		e.stopCmd(containerName),
		e.removeCmd(containerName),
	)

	return cmds, nil
}

func (e *dockerEngine) CreateVolumeCmds(t *jobs.Task, vol jobs.Volume, volIndex int, volPrefix, volHostPath string, env map[string]string, scalingGroup uint) (engine.Cmds, error) {
	containerImage := e.images.CephVolume
	containerName := createVolumeUnitContainerName(t, volIndex, scalingGroup)
	execStart, err := e.createVolumeDockerCmdLine(t, containerName, containerImage, vol, volPrefix, volHostPath, env, scalingGroup)
	if err != nil {
		return engine.Cmds{}, maskAny(err)
	}
	testVolHostPathCmd := *cmdline.New(nil, "/bin/sh", "-c", fmt.Sprintf("'test -e %s || mkdir -p %s'", volHostPath, volHostPath))

	var cmds engine.Cmds
	cmds.Start = append(cmds.Start,
		e.pullCmd(containerImage),
		testVolHostPathCmd,
		e.stopCmd(containerName),
		e.removeCmd(containerName),
		e.cleanupCmd(),
	)
	if e.options.EnvFile != "" {
		cmds.Start = append(cmds.Start, *cmdline.New(nil, "/usr/bin/touch", e.options.EnvFile))
	}
	cmds.Start = append(cmds.Start, execStart)

	cmds.Stop = append(cmds.Stop,
		e.stopCmd(containerName),
		e.removeCmd(containerName),
	)

	return cmds, nil
}

func (e *dockerEngine) pullCmd(image string) cmdline.Cmdline {
	return *cmdline.New(nil, e.dockerPath, "pull", image)
}

func (e *dockerEngine) stopCmd(containerName string) cmdline.Cmdline {
	cmd := cmdline.Cmdline{AllowFailure: true}
	cmd.Add(nil, e.dockerPath, "stop", fmt.Sprintf("-t %v", e.containerTimeoutStopSec), containerName)
	return cmd
}

func (e *dockerEngine) removeCmd(containerName string) cmdline.Cmdline {
	cmd := cmdline.Cmdline{AllowFailure: true}
	cmd.Add(nil, e.dockerPath, "rm", "-f", containerName)
	return cmd
}

func (e *dockerEngine) cleanupCmd() cmdline.Cmdline {
	cmd := cmdline.Cmdline{AllowFailure: true}
	cmd.Add(nil, e.cleanupScriptPath)
	return cmd
}

// createVolumeUnitContainerName creates the name of the docker container that serves a volume with given index
func createVolumeUnitContainerName(t *jobs.Task, volIndex int, scalingGroup uint) string {
	return fmt.Sprintf("%s-vl%d", t.ContainerName(scalingGroup), volIndex)
}
