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

package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/juju/errgo"
	"github.com/spf13/cobra"

	fg "git.pulcy.com/pulcy/deployit/flags"
	"git.pulcy.com/pulcy/deployit/fleet"
	"git.pulcy.com/pulcy/deployit/units"
)

var (
	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Create or update a job on a stack.",
		Long:  "Create or update a job on a stack.",
		Run:   runRun,
	}
	runFlags struct {
		fg.Flags
	}
)

func init() {
	initDeploymentFlags(runCmd.Flags(), &runFlags.Flags)
}

func runRun(cmd *cobra.Command, args []string) {
	ctx := units.RenderContext{
		ProjectName:    cmdMain.Use,
		ProjectVersion: projectVersion,
		ProjectBuild:   projectBuild,
	}

	deploymentDefaults(cmd.Flags(), &runFlags.Flags, args)
	runValidators(&runFlags.Flags)

	cluster, err := loadCluster(&runFlags.Flags)
	if err != nil {
		Exitf("Cannot load cluster: %v\n", err)
	}
	job, err := loadJob(&runFlags.Flags, *cluster)
	if err != nil {
		Exitf("Cannot load job: %v\n", err)
	}

	groups := groups(&runFlags.Flags)
	generator := job.Generate(groups, runFlags.ScalingGroup)
	assert(generator.WriteTmpFiles(ctx, images, cluster.InstanceCount))

	if runFlags.DryRun {
		confirm(fmt.Sprintf("remove tmp files from %s ?", generator.TmpDir()))
	} else {
		location := cluster.Stack
		count := job.MaxCount()
		updateScalingGroups(&runFlags.ScalingGroup, count, location, func(runUpdate runUpdateCallback) {
			generator := job.Generate(groups, runFlags.ScalingGroup)

			assert(generator.WriteTmpFiles(ctx, images, cluster.InstanceCount))

			unitNames := generator.UnitNames()
			fileNames := generator.FileNames()

			runUpdate(cluster.Stack, cluster.Tunnel, unitNames, fileNames, runFlags.StopDelay, runFlags.DestroyDelay, runFlags.Force)
			assert(generator.RemoveTmpFiles())
		}, runFlags.SliceDelay, runFlags.Force)
	}
}

func runValidators(f *fg.Flags) {
}

func doRunUpdate(stack, tunnel string, unitNames, files []string, stopDelay, destroyDelay time.Duration, force bool) {
	if len(unitNames) != len(files) {
		panic("Internal update error")
	}

	f := fleet.NewTunnel(tunnel)
	loadedUnitNames, err := selectLoadedUnits(unitNames, f)
	assert(err)
	modifiedUnitNames, err := selectModifiedUnits(loadedUnitNames, files, f, force)
	assert(err)
	if len(modifiedUnitNames) > 0 {
		assert(destroyUnits(stack, f, modifiedUnitNames, stopDelay))
		fmt.Printf("Waiting for %s...\n", destroyDelay)
		time.Sleep(destroyDelay)
	}

	assert(launchUnits(tunnel, files))
}

// selectLoadedUnits filters the given list of unit names, leaving in
// only the units that are loaded on in fleet.
func selectLoadedUnits(unitNames []string, tunnel *fleet.FleetTunnel) ([]string, error) {
	list, err := tunnel.List()
	if err != nil {
		return nil, maskAny(err)
	}
	result := []string{}
	for _, name := range unitNames {
		if contains(list, name) {
			result = append(result, name)
		}
	}
	return result, nil
}

// selectModifiedUnits filters the given list of unit names, leaving in
// only those units that are actually different than in fleet.
func selectModifiedUnits(unitNames, files []string, tunnel *fleet.FleetTunnel, force bool) ([]string, error) {
	if force {
		return unitNames, nil
	}
	result := []string{}
	for _, unitName := range unitNames {
		cat, err := tunnel.Cat(unitName)
		if err != nil {
			fmt.Printf("Failed to cat '%s': %#v\n", unitName, err)
			result = append(result, unitName)
			continue
		}
		newCat, err := readUnit(unitName, files)
		if err != nil {
			return nil, maskAny(err)
		}
		if !compareUnitContent(cat, newCat) {
			result = append(result, unitName)
		} else {
			fmt.Printf("Unit '%s' has not changed\n", unitName)
		}
	}
	return result, nil
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

func contains(list []string, value string) bool {
	for _, x := range list {
		if x == value {
			return true
		}
	}
	return false
}

type runUpdateCallback func(stack, tunnel string, unitNames, files []string, stopDelay, destroyDelay time.Duration, force bool)

// updateScalingGroups calls a given update function for each scaling group, such that they all update in succession.
// confirmation is asked before updating more than 1 scaling group
func updateScalingGroups(scalingGroup *uint, scale uint, stack string, updateCurrentGroup func(runUpdate runUpdateCallback), sliceDelay time.Duration, force bool) {
	if *scalingGroup != 0 {
		// Only one group to update
		updateCurrentGroup(doRunUpdate)
	} else {
		// Detect how many scaling groups there actually are (yield units names)
		maxScale, unitNames := detectLargestScalingGroup(scalingGroup, scale, updateCurrentGroup)

		// Ask for confirmation
		if !force {
			var confirmMsg string
			confirmMsg = fmt.Sprintf("Are you sure you want to update stack '%s' scaling groups 1-%d?\nUnits:\n- %s\nEnter yes:", stack, maxScale, strings.Join(unitNames, "\n- "))
			if err := confirm(confirmMsg); err != nil {
				panic(err)
			}
			fmt.Println()
		}

		// Update all scaling groups in successions
		for sg := uint(1); sg <= maxScale; sg++ {
			// Tell what we're going to do
			fmt.Printf("Updating scaling group %d of %d...\n", sg, maxScale)

			// Set current scaling group
			*scalingGroup = sg

			// Call update function
			updateCurrentGroup(doRunUpdate)

			// Wait a bit and ask for confirmation before continuing (only when more groups will follow)
			if sg < maxScale {
				fmt.Printf("Waiting %s before continuing with scaling group %d of %d...\n", sliceDelay, sg+1, maxScale)
				time.Sleep(sliceDelay)

				// Ask for confirmation to continue
				if !force {
					if err := confirm(fmt.Sprintf("Are you sure to continue with scaling group %d of %d on '%s'?\nEnter yes:", sg+1, maxScale, stack)); err != nil {
						panic(err)
					}
				}
			}
		}
	}
}

// detectLargestScalingGroup tries to detect the largest scaling group that still yields units.
func detectLargestScalingGroup(scalingGroup *uint, defaultScale uint, updateCurrentGroup func(runUpdate runUpdateCallback)) (uint, []string) {
	// Start with 2 since we assume there is always at least 1 scaling group
	var names []string
	*scalingGroup = 1
	updateCurrentGroup(func(stack, tunnel string, unitNames, files []string, stopDelay, destroyDelay time.Duration, force bool) {
		names = unitNames
	})
	for sg := uint(2); sg <= defaultScale; sg++ {
		// Set current scaling group
		*scalingGroup = sg
		var hasUnits bool
		updateCurrentGroup(func(stack, tunnel string, unitNames, files []string, stopDelay, destroyDelay time.Duration, force bool) {
			hasUnits = len(unitNames) > 0
		})
		if !hasUnits {
			return sg - 1, names
		}
	}
	return defaultScale, names
}

func launchUnits(tunnel string, files []string) error {
	f := fleet.NewTunnel(tunnel)

	Verbosef("Starting %#v\n", files)
	out, err := f.Start(files...)
	if err != nil {
		return errgo.Mask(err)
	}

	fmt.Println(out)
	return nil
}
