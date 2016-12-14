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

package extpoints

import (
	"github.com/pulcy/j2/cluster"
	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/render"
	"github.com/pulcy/j2/scheduler"
)

type Orchestrator interface {
	// RenderProvider returns the provider for the unit renderer for this orchestrator.
	RenderProvider() (render.RenderProvider, error)

	// Scheduler returns the scheduler, configured for the given cluster, for this orchestrator.
	Scheduler(jobs.Job, cluster.Cluster) (scheduler.Scheduler, error)
}
