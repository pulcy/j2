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
	"github.com/pulcy/j2/cluster"
	"github.com/pulcy/j2/engine"
	"github.com/pulcy/j2/extpoints"
)

type dockerEngineProvider struct{}

func (p *dockerEngineProvider) NewEngine(cluster cluster.Cluster) engine.Engine {
	return &dockerEngine{
		options:                 cluster.DockerOptions,
		dockerPath:              "/usr/bin/docker",
		shPath:                  "/bin/sh",
		touchPath:               "/usr/bin/touch",
		cleanupScriptPath:       "/home/core/bin/docker-cleanup.sh",
		weavePluginSocket:       "unix:///var/run/weave/weave.sock",
		containerTimeoutStopSec: 10,
	}
}

func init() {
	extpoints.EngineProviders.Register(&dockerEngineProvider{}, "docker")
}
