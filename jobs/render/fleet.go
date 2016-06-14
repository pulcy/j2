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

package render

import (
	"github.com/pulcy/j2/cluster"
	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/pkg/sdunits"
)

// CreateFleetAfter creates a list of unit names to add to the `After` setting of each unit of the given task.
func addFleetOptions(t *jobs.Task, fleetOptions cluster.FleetOptions, unit *sdunits.Unit) {
	jobName := t.JobName()
	unit.ExecOptions.After(excludeUnitsOfJob(jobName, fleetOptions.After)...)
	unit.ExecOptions.Want(excludeUnitsOfJob(jobName, fleetOptions.Wants)...)
	unit.ExecOptions.Require(excludeUnitsOfJob(jobName, fleetOptions.Requires)...)
}

// excludeUnitsOfJob creates a copy of the given unit names list, excluding those unit names that
// belong to the given job.
func excludeUnitsOfJob(jobName jobs.JobName, unitNames []string) []string {
	result := []string{}
	for _, unitName := range unitNames {
		if !jobs.IsUnitForJob(unitName, jobName) {
			result = append(result, unitName)
		}
	}
	return result
}
