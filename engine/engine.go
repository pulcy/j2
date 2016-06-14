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

package engine

import (
	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/pkg/cmdline"
)

type Cmds struct {
	Start []cmdline.Cmdline
	Stop  []cmdline.Cmdline
}

type Engine interface {
	CreateMainCmds(t *jobs.Task, env map[string]string, scalingGroup uint) (Cmds, error)
	CreateProxyCmds(t *jobs.Task, link jobs.Link, linkIndex int, env map[string]string, scalingGroup uint) (Cmds, error)
	CreateVolumeCmds(t *jobs.Task, vol jobs.Volume, volIndex int, volPrefix, volHostPath string, env map[string]string, scalingGroup uint) (Cmds, error)
}
