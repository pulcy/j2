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

package jobs

import (
	"github.com/pulcy/j2/cluster"
	"github.com/pulcy/j2/pkg/sdunits"
)

// CreateFleetAfter creates a list of unit names to add to the `After` setting of each unit of the given task.
func (t *Task) AddFleetOptions(fleetOptions cluster.FleetOptions, unit *sdunits.Unit) {
	unit.ExecOptions.After(t.group.job.excludeUnitsOfJob(fleetOptions.After)...)
	unit.ExecOptions.Want(t.group.job.excludeUnitsOfJob(fleetOptions.Wants)...)
	unit.ExecOptions.Require(t.group.job.excludeUnitsOfJob(fleetOptions.Requires)...)
}

// excludeUnitsOfJob creates a copy of the given unit names list, excluding those unit names that
// belong to the given job.
func (j *Job) excludeUnitsOfJob(unitNames []string) []string {
	result := []string{}
	for _, unitName := range unitNames {
		if !IsUnitForJob(unitName, j.Name) {
			result = append(result, unitName)
		}
	}
	return result
}
