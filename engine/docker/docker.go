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
	shPath                  string
	touchPath               string
	cleanupScriptPath       string
	weavePluginSocket       string
	containerTimeoutStopSec int
}

// NewDockerEngine creates a new engine renderer for docker
func NewDockerEngine(options cluster.DockerOptions) engine.Engine {
	return &dockerEngine{
		options:                 options,
		dockerPath:              "/usr/bin/docker",
		shPath:                  "/bin/sh",
		touchPath:               "/usr/bin/touch",
		cleanupScriptPath:       "/home/core/bin/docker-cleanup.sh",
		weavePluginSocket:       "unix:///var/run/weave/weave.sock",
		containerTimeoutStopSec: 10,
	}
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

// createDockerCmd creates a docker command suitable for the given network type.
func (e *dockerEngine) createDockerCmd(env map[string]string, networkType jobs.NetworkType) (cmdline.Cmdline, error) {
	c := cmdline.Cmdline{}
	switch networkType {
	case "", jobs.NetworkTypeDefault, jobs.NetworkTypeHost:
		return *c.Add(nil, e.dockerPath), nil
	case jobs.NetworkTypeWeave:
		return *c.Add(nil, e.dockerPath).Add(env, fmt.Sprintf("-H=%s", e.weavePluginSocket)), nil
	default:
		return cmdline.Cmdline{}, maskAny(fmt.Errorf("Unknown network type '%s", networkType))
	}
}

// addDockerNetworkArgs adds docker network arguments for the given task.
func (e *dockerEngine) addDockerNetworkArgs(c *cmdline.Cmdline, env map[string]string, t *jobs.Task) error {
	switch t.Network {
	case "", jobs.NetworkTypeDefault:
		return nil
	case jobs.NetworkTypeHost:
		c.Add(env, "--net=host")
		return nil
	case jobs.NetworkTypeWeave:
		if !t.Type.IsProxy() {
			c.Add(env, fmt.Sprintf("--hostname=%s", t.WeaveDomainName()))
		} else {
			c.Add(env, fmt.Sprintf("--hostname=%s", t.PrivateDomainName()))
		}
		return nil
	default:
		return maskAny(fmt.Errorf("Unknown network type '%s", t.Network))
	}
}

// createVolumeUnitContainerName creates the name of the docker container that serves a volume with given index
func createVolumeUnitContainerName(t *jobs.Task, volIndex int, scalingGroup uint) string {
	return fmt.Sprintf("%s-vl%d", t.ContainerName(scalingGroup), volIndex)
}
