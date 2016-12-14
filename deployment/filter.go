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
	"github.com/pulcy/j2/scheduler"
)

// createUnitNameFilter creates a predicate that return true if a given
// unit name belongs to the configured job and task-group selection.
func (d *Deployment) createUnitNamePredicate() func(scheduler.Unit) bool {
	if d.groupSelection.IncludeAll() {
		// Select everything in the job
		return func(unit scheduler.Unit) bool {
			if !d.scalingGroupSelection.IncludeAll() {
				if !jobs.IsUnitForScalingGroup(unit.Name(), d.job.Name, uint(d.scalingGroupSelection)) {
					return false
				}
			}
			return jobs.IsUnitForJob(unit.Name(), d.job.Name)
		}
	}

	// Select everything in one of the groups
	return func(unit scheduler.Unit) bool {
		if !d.scalingGroupSelection.IncludeAll() {
			if !jobs.IsUnitForScalingGroup(unit.Name(), d.job.Name, uint(d.scalingGroupSelection)) {
				return false
			}
		}
		for _, g := range d.groupSelection {
			if jobs.IsUnitForTaskGroup(unit.Name(), d.job.Name, g) {
				return true
			}
		}
		return false
	}
}

// selectUnitNames returns al unit files from the given list where the given predicate returns true.
func selectUnitNames(units []scheduler.Unit, predicate func(scheduler.Unit) bool) []scheduler.Unit {
	result := []scheduler.Unit{}
	for _, x := range units {
		if predicate(x) {
			result = append(result, x)
		}
	}
	return result
}

// containsPredicate creates a predicate that returns true if the given unitName is contained
// in the given list.
func containsPredicate(list []scheduler.Unit) func(scheduler.Unit) bool {
	return func(unit scheduler.Unit) bool {
		for _, x := range list {
			if x.Name() == unit.Name() {
				return true
			}
		}
		return false
	}
}

func notPredicate(predicate func(scheduler.Unit) bool) func(scheduler.Unit) bool {
	return func(unit scheduler.Unit) bool {
		return !predicate(unit)
	}
}
