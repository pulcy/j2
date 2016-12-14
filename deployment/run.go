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
	"fmt"
	"sort"
	"strings"

	"github.com/ryanuber/columnize"

	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/scheduler"
)

// Run creates all applicable unit files and deploys them onto the configured cluster.
func (d *Deployment) Run() error {
	// Fetch all current units
	s, err := d.orchestrator.Scheduler(d.cluster)
	if err != nil {
		return maskAny(err)
	}

	allUnits, err := s.List()
	if err != nil {
		return maskAny(err)
	}

	// Prepare UI
	ui := newStateUI(d.verbose)
	defer ui.Close()

	// Find out which current units belong to the configured job
	remainingLoadedJobUnitNames := selectUnitNames(allUnits, d.createUnitNamePredicate())

	// Create scaling group units
	if err := d.generateScalingGroups(); err != nil {
		return maskAny(err)
	}

	// Ask for confirmation
	maxScale := d.scalingGroups[len(d.scalingGroups)-1].scalingGroup

	// Go over every scale
	step := 1
	totalModifications := 0
	waitBeforeNextStep := false
	for sgIndex, sg := range d.scalingGroups {
		// Select the loaded units that belong to this scaling group
		correctScalingGroupPredicate := func(unit scheduler.Unit) bool {
			return jobs.IsUnitForScalingGroup(unit.Name(), d.job.Name, sg.scalingGroup)
		}
		loadedScalingGroupUnitNames := selectUnitNames(remainingLoadedJobUnitNames, correctScalingGroupPredicate)
		// Update remainingLoadedJobUnitNames
		remainingLoadedJobUnitNames = selectUnitNames(remainingLoadedJobUnitNames, notPredicate(containsPredicate(loadedScalingGroupUnitNames)))

		// Select the loaded unit name that have become obsolete
		sgUnitNames := sg.Units()
		obsoleteUnitNames := selectUnitNames(loadedScalingGroupUnitNames, notPredicate(containsPredicate(sgUnitNames)))
		notObsoleteUnitNames := selectUnitNames(loadedScalingGroupUnitNames, containsPredicate(sgUnitNames))

		// Select the unit names that are modified and need an update
		isModifiedPredicate := d.isModifiedPredicate(sg, s, ui)
		modifiedUnitNames := selectUnitNames(notObsoleteUnitNames, isModifiedPredicate)
		isFailedPredicate := d.isFailedPredicate(sg, s, ui)
		failedUnitNames := selectUnitNames(notObsoleteUnitNames, isFailedPredicate)
		unitNamesToDestroy := append(append(obsoleteUnitNames, modifiedUnitNames...), failedUnitNames...)
		newUnitNames := selectUnitNames(sgUnitNames, notPredicate(containsPredicate(loadedScalingGroupUnitNames)))

		// Are there any changes?
		anyModifications := (len(loadedScalingGroupUnitNames) != len(sg.units)) || (len(unitNamesToDestroy) > 0)

		// Wait a bit before proceeding
		if waitBeforeNextStep && anyModifications {
			InterruptibleSleep(ui.MessageSink, d.SliceDelay, fmt.Sprintf("Waiting %s before continuing with scaling group %d of %d...", "%s", (sgIndex+1), maxScale))
			ui.Clear()
		}

		// Confirm modifications
		if anyModifications && !d.force {
			curScale := sg.scalingGroup
			changes := []string{"# Unit | Action"}
			changes = append(changes, formatChanges("# ", unitsToNames(obsoleteUnitNames), "Remove (is obsolete) !!!")...)
			changes = append(changes, formatChanges("# ", unitsToNames(modifiedUnitNames), "Update")...)
			changes = append(changes, formatChanges("# ", unitsToNames(failedUnitNames), "Failed state")...)
			changes = append(changes, formatChanges("# ", unitsToNames(newUnitNames), "Create")...)
			sort.Strings(changes[1:])
			formattedChanges := strings.Replace(columnize.SimpleFormat(changes), "#", " ", -1)
			ui.HeaderSink <- fmt.Sprintf("Step %d: Update scaling group %d of %d on '%s'.\n%s\n", step, curScale, maxScale, d.cluster.Stack, formattedChanges)
			if !d.autoContinue {
				if err := ui.Confirm("Are you sure you want to continue?"); err != nil {
					return maskAny(err)
				}
			}
		}

		// Destroy the obsolete & modified units
		if len(unitNamesToDestroy) > 0 {
			if err := d.destroyUnits(s, unitNamesToDestroy, ui); err != nil {
				return maskAny(err)
			}

			InterruptibleSleep(ui.MessageSink, d.DestroyDelay, "Waiting for %s...")
		}

		// Now launch everything
		unitsToLaunch := sg.selectByNames(modifiedUnitNames, failedUnitNames, newUnitNames)
		if unitsToLaunch.Len() > 0 {
			if err := launchUnits(s, unitsToLaunch, ui); err != nil {
				return maskAny(err)
			}
		}

		// Update counters
		if anyModifications {
			waitBeforeNextStep = true
			totalModifications++
		}
		step++
		ui.Clear()
	}

	// Destroy remaining units
	if len(remainingLoadedJobUnitNames) > 0 {
		changes := []string{"# Unit | Action"}
		changes = append(changes, formatChanges("# ", unitsToNames(remainingLoadedJobUnitNames), "Remove (is obsolete) !!!")...)
		sort.Strings(changes[1:])
		formattedChanges := strings.Replace(columnize.SimpleFormat(changes), "#", " ", -1)
		ui.HeaderSink <- fmt.Sprintf("Step %d: Cleanup of obsolete units on '%s'.\n%s\n", step, d.cluster.Stack, formattedChanges)
		if err := ui.Confirm("Are you sure you want to continue?"); err != nil {
			return maskAny(err)
		}

		if err := d.destroyUnits(s, remainingLoadedJobUnitNames, ui); err != nil {
			return maskAny(err)
		}

		totalModifications++
	}

	// Notify in case we did nothing
	if totalModifications == 0 {
		ui.MessageSink <- "No modifications needed."
	} else {
		ui.MessageSink <- "Done."
	}

	return nil
}

// isFailedPredicate creates a predicate that returns true when the given unit file is in the failed status.
func (d *Deployment) isFailedPredicate(sg scalingGroupUnits, f scheduler.Scheduler, ui *stateUI) func(scheduler.Unit) bool {
	return func(unit scheduler.Unit) bool {
		ui.MessageSink <- fmt.Sprintf("Checking state of %s", unit.Name())
		unitState, err := f.GetState(unit)
		if scheduler.IsNotFound(err) {
			ui.Verbosef("Unit '%s' is not found\n", unit.Name())
			return true
		} else if err != nil {
			ui.Warningf("GetState(%s) failed: %#v", unit.Name(), err)
		}
		if unitState.Failed {
			ui.Verbosef("Unit '%s' is in failed state\n", unit.Name())
			return true
		}
		return false
	}
}

// isModifiedPredicate creates a predicate that returns true when the given unit file is modified
func (d *Deployment) isModifiedPredicate(sg scalingGroupUnits, f scheduler.Scheduler, ui *stateUI) func(scheduler.Unit) bool {
	return func(unit scheduler.Unit) bool {
		if d.force {
			return true
		}
		ui.MessageSink <- fmt.Sprintf("Checking %s for modifications", unit.Name())
		cat, err := f.Cat(unit)
		if err != nil {
			ui.Verbosef("Failed to cat '%s': %#v\n", unit.Name(), err)
			return true // Assume it is modified
		}
		newUnit, err := sg.get(unit)
		if err != nil {
			ui.Verbosef("Failed to read new '%s' unit: %#v\n", unit.Name(), err)
			return true // Assume it is modified
		}
		if !compareUnitContent(unit.Name(), cat, newUnit.Content(), ui) {
			return true
		}
		ui.Verbosef("Unit '%s' has not changed\n", unit.Name())
		return false
	}
}

func compareUnitContent(unitName, a, b string, ui *stateUI) bool {
	linesA := normalizeUnitContent(a)
	linesB := normalizeUnitContent(b)

	if len(linesA) != len(linesB) {
		ui.Verbosef("Length differs in %s\n", unitName)
		return false
	}
	for i, la := range linesA {
		lb := linesB[i]
		if la != lb {
			ui.Verbosef("Line %d in %s differs\n>>>> %s\n<<<< %s\n", i, unitName, la, lb)
			return false
		}
	}
	return true
}

func normalizeUnitContent(content string) []string {
	lines := strings.Split(content, "\n")
	result := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func launchUnits(f scheduler.Scheduler, units scheduler.UnitDataList, ui *stateUI) error {
	ui.Verbosef("Starting %#v\n", units)

	ui.MessageSink <- fmt.Sprintf("Starting %d unit(s)", units.Len())
	if err := f.Start(ui.EventSink, units); err != nil {
		return maskAny(err)
	}

	return nil
}

func formatChanges(prefix string, unitNames []string, action string) []string {
	result := []string{}
	for _, x := range unitNames {
		result = append(result, fmt.Sprintf("%s%s | %s", prefix, x, action))
	}
	sort.Strings(result)
	return result
}
