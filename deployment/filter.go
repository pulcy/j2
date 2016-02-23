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

package deployment

import (
	"github.com/pulcy/j2/jobs"
)

// createUnitNameFilter creates a predicate that return true if a given
// unit name belongs to the configured job and task-group selection.
func (d *Deployment) createUnitNamePredicate() func(string) bool {
	if d.groupSelection.IncludeAll() {
		// Select everything in the job
		return func(unitName string) bool {
			if !d.scalingGroupSelection.IncludeAll() {
				if !jobs.IsUnitForScalingGroup(unitName, d.job.Name, uint(d.scalingGroupSelection)) {
					return false
				}
			}
			return jobs.IsUnitForJob(unitName, d.job.Name)
		}
	}

	// Select everything in one of the groups
	return func(unitName string) bool {
		if !d.scalingGroupSelection.IncludeAll() {
			if !jobs.IsUnitForScalingGroup(unitName, d.job.Name, uint(d.scalingGroupSelection)) {
				return false
			}
		}
		for _, g := range d.groupSelection {
			if jobs.IsUnitForTaskGroup(unitName, d.job.Name, g) {
				return true
			}
		}
		return false
	}
}

// selectUnitNames returns al unit files from the given list where the given predicate returns true.
func selectUnitNames(unitNames []string, predicate func(string) bool) []string {
	result := []string{}
	for _, x := range unitNames {
		if predicate(x) {
			result = append(result, x)
		}
	}
	return result
}

// containsPredicate creates a predicate that returns true if the given unitName is contained
// in the given list.
func containsPredicate(list []string) func(string) bool {
	return func(unitName string) bool {
		for _, x := range list {
			if x == unitName {
				return true
			}
		}
		return false
	}
}

func notPredicate(predicate func(string) bool) func(string) bool {
	return func(unitName string) bool {
		return !predicate(unitName)
	}
}
