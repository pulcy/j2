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
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ryanuber/columnize"

	"github.com/pulcy/j2/fleet"
	"github.com/pulcy/j2/jobs"
)

// Run creates all applicable unit files and deploys them onto the configured cluster.
func (d *Deployment) Run(deps DeploymentDependencies) error {
	// Fetch all current units
	f := d.newFleetTunnel()
	allUnits, err := f.List()
	if err != nil {
		return maskAny(err)
	}

	// Find out which current units belong to the configured job
	remainingLoadedJobUnitNames := selectUnitNames(allUnits, d.createUnitNamePredicate())

	// Create scaling group units
	defer d.cleanup()
	if err := d.generateScalingGroups(); err != nil {
		return maskAny(err)
	}

	// Ask for confirmation
	maxScale := d.scalingGroups[len(d.scalingGroups)-1].scalingGroup

	// Go over every scale
	step := 1
	totalModifications := 0
	for sgIndex, sg := range d.scalingGroups {
		// Select the loaded units that belong to this scaling group
		correctScalingGroupPredicate := func(unitName string) bool {
			return jobs.IsUnitForScalingGroup(unitName, d.job.Name, sg.scalingGroup)
		}
		loadedScalingGroupUnitNames := selectUnitNames(remainingLoadedJobUnitNames, correctScalingGroupPredicate)
		// Update remainingLoadedJobUnitNames
		remainingLoadedJobUnitNames = selectUnitNames(remainingLoadedJobUnitNames, notPredicate(containsPredicate(loadedScalingGroupUnitNames)))

		// Select the loaded unit name that have become obsolete
		obsoleteUnitNames := selectUnitNames(loadedScalingGroupUnitNames, notPredicate(containsPredicate(sg.unitNames)))
		notObsoleteUnitNames := selectUnitNames(loadedScalingGroupUnitNames, containsPredicate(sg.unitNames))

		// Select the unit names that are modified and need an update
		statusMap, err := f.Status()
		if err != nil {
			return maskAny(err)
		}
		isModifiedPredicate := d.isModifiedOrFailedPredicate(deps, sg, statusMap, f)
		modifiedUnitNames := selectUnitNames(notObsoleteUnitNames, isModifiedPredicate)
		unitNamesToDestroy := append(obsoleteUnitNames, modifiedUnitNames...)
		newUnitNames := selectUnitNames(sg.unitNames, notPredicate(containsPredicate(loadedScalingGroupUnitNames)))

		// Are there any changes?
		anyModifications := (len(loadedScalingGroupUnitNames) != len(sg.fileNames)) || (len(unitNamesToDestroy) > 0)

		// Confirm modifications
		if anyModifications && !d.force {
			curScale := sg.scalingGroup
			changes := []string{"# Unit | Action"}
			changes = append(changes, formatChanges("# ", obsoleteUnitNames, "Remove (is obsolete) !!!")...)
			changes = append(changes, formatChanges("# ", modifiedUnitNames, "Update")...)
			changes = append(changes, formatChanges("# ", newUnitNames, "Create")...)
			sort.Strings(changes[1:])
			formattedChanges := strings.Replace(columnize.SimpleFormat(changes), "#", " ", -1)
			fmt.Printf("Step %d: Update scaling group %d of %d on '%s'.\n%s\n", step, curScale, maxScale, d.cluster.Stack, formattedChanges)
			if err := deps.Confirm("Are you sure you want to continue?"); err != nil {
				return maskAny(err)
			}
			fmt.Println()
		}

		// Destroy the obsolete & modified units
		if len(unitNamesToDestroy) > 0 {
			if err := d.destroyUnits(f, unitNamesToDestroy); err != nil {
				return maskAny(err)
			}

			fmt.Printf("Waiting for %s...\n", d.DestroyDelay)
			time.Sleep(d.DestroyDelay)
		}

		// Now launch everything
		if err := launchUnits(deps, f, sg.fileNames); err != nil {
			return maskAny(err)
		}

		// Wait a bit and ask for confirmation before continuing (only when more groups will follow)
		if anyModifications && sgIndex+1 < len(d.scalingGroups) {
			nextScale := d.scalingGroups[sgIndex+1].scalingGroup
			fmt.Printf("Waiting %s before continuing with scaling group %d of %d...\n", d.SliceDelay, nextScale, maxScale)
			time.Sleep(d.SliceDelay)
		}

		// Update counters
		if anyModifications {
			totalModifications++
		}
		step++
	}

	// Destroy remaining units
	if len(remainingLoadedJobUnitNames) > 0 {
		changes := []string{"# Unit | Action"}
		changes = append(changes, formatChanges("# ", remainingLoadedJobUnitNames, "Remove (is obsolete) !!!")...)
		sort.Strings(changes[1:])
		formattedChanges := strings.Replace(columnize.SimpleFormat(changes), "#", " ", -1)
		fmt.Printf("Step %d: Cleanup of obsolete units on '%s'.\n%s\n", step, d.cluster.Stack, formattedChanges)
		if err := deps.Confirm("Are you sure you want to continue?"); err != nil {
			return maskAny(err)
		}
		fmt.Println()

		if err := d.destroyUnits(f, remainingLoadedJobUnitNames); err != nil {
			return maskAny(err)
		}

		totalModifications++
	}

	// Notify in case we did nothing
	if totalModifications == 0 {
		fmt.Printf("No modifications needed.\n")
	} else {
		fmt.Printf("Done.\n")
	}

	return nil
}

// isModifiedOrFailedPredicate creates a predicate that returns true when the given unit file is modified or its status
// is failed.
func (d *Deployment) isModifiedOrFailedPredicate(deps DeploymentDependencies, sg scalingGroupUnits, status fleet.StatusMap, f fleet.FleetTunnel) func(string) bool {
	return func(unitName string) bool {
		if d.force {
			return true
		}
		unitState, found := status.Get(unitName)
		if unitState == "failed" || !found {
			return true
		}
		cat, err := f.Cat(unitName)
		if err != nil {
			fmt.Printf("Failed to cat '%s': %#v\n", unitName, err)
			return true // Assume it is modified
		}
		newCat, err := readUnit(unitName, sg.fileNames)
		if err != nil {
			fmt.Printf("Failed to read new '%s' unit: %#v\n", unitName, err)
			return true // Assume it is modified
		}
		if !compareUnitContent(cat, newCat) {
			return true
		}
		deps.Verbosef("Unit '%s' has not changed\n", unitName)
		return false
	}
}

func readUnit(unitName string, files []string) (string, error) {
	for _, fileName := range files {
		if unitName != filepath.Base(fileName) {
			continue
		}
		data, err := ioutil.ReadFile(fileName)
		if err != nil {
			return "", maskAny(err)
		}
		return string(data), nil
	}
	return "", nil // This will ensure that the unit is considered different
}

func compareUnitContent(a, b string) bool {
	a = normalizeUnitContent(a)
	b = normalizeUnitContent(b)
	return a == b
}

func normalizeUnitContent(content string) string {
	lines := strings.Split(content, "\n")
	result := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

func launchUnits(deps DeploymentDependencies, f fleet.FleetTunnel, files []string) error {
	deps.Verbosef("Starting %#v\n", files)
	out, err := f.Start(files...)
	if err != nil {
		return maskAny(err)
	}

	if out != "" {
		fmt.Println(out)
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
