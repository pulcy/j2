package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	fg "arvika.pulcy.com/pulcy/deployit/flags"
	"arvika.pulcy.com/pulcy/deployit/fleet"
)

var (
	updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Update a job on a stack.",
		Long:  "Update a job on a stack.",
		Run:   updateRun,
	}
	updateFlags struct {
		fg.Flags
	}
)

func init() {
	initDeploymentFlags(updateCmd.Flags(), &updateFlags.Flags)
}

func updateRun(cmd *cobra.Command, args []string) {
	deploymentDefaults(&updateFlags.Flags, args)
	createValidators(&updateFlags.Flags)
	deploymentValidators(&updateFlags.Flags)

	job, err := loadJob(&updateFlags.Flags)
	if err != nil {
		Exitf("Cannot load job: %v\n", err)
	}
	groups := groups(&updateFlags.Flags)
	generator := job.Generate(groups, updateFlags.ScalingGroup)
	assert(generator.WriteTmpFiles())

	if updateFlags.DryRun {
		confirm(fmt.Sprintf("remove tmp files from %s ?", generator.TmpDir()))
	} else {
		location := updateFlags.Stack
		count := job.MaxCount()
		updateScalingGroups(&updateFlags.ScalingGroup, count, location, func(runUpdate runUpdateCallback) {
			generator := job.Generate(groups, updateFlags.ScalingGroup)

			assert(generator.WriteTmpFiles())

			unitNames := generator.UnitNames()
			fileNames := generator.FileNames()

			runUpdate(updateFlags.Stack, updateFlags.Tunnel, unitNames, fileNames, updateFlags.StopDelay, updateFlags.DestroyDelay)
			assert(generator.RemoveTmpFiles())
		}, updateFlags.SliceDelay)
	}
}

func doRunUpdate(stack, tunnel string, unitNames, files []string, stopDelay, destroyDelay time.Duration) {
	if len(unitNames) != len(files) {
		panic("Internal update error")
	}

	f := fleet.NewTunnel(tunnel)
	assert(destroyUnits(stack, f, unitNames, stopDelay))

	fmt.Printf("Waiting for %s seconds ...\n", destroyDelay)
	time.Sleep(destroyDelay)

	assert(createUnits(tunnel, files))
}

type runUpdateCallback func(stack, tunnel string, unitNames, files []string, stopDelay, destroyDelay time.Duration)

// updateScalingGroups calls a given update function for each scaling group, such that they all update in succession.
// confirmation is asked before updating more than 1 scaling group
func updateScalingGroups(scalingGroup *uint, scale uint, stack string, updateCurrentGroup func(runUpdate runUpdateCallback), sliceDelay time.Duration) {
	if *scalingGroup != 0 {
		// Only one group to update
		updateCurrentGroup(doRunUpdate)
	} else {
		// Detect how many scaling groups there actually are (yield units names)
		maxScale := detectLargestScalingGroup(scalingGroup, scale, updateCurrentGroup)

		// Ask for confirmation
		var confirmMsg string
		if maxScale == 1 {
			confirmMsg = fmt.Sprintf("Are you sure you want to update '%s'? Enter yes:", stack)
		} else {
			confirmMsg = fmt.Sprintf("Are you sure you want to update all %v scaling groups one after another on '%s'? Enter yes:", maxScale, stack)
		}
		if err := confirm(confirmMsg); err != nil {
			panic(err)
		}

		// Update all scaling groups in successions
		for sg := uint(1); sg <= maxScale; sg++ {
			// Tell what we're going to do
			fmt.Printf("Updating scaling group %v ...\n", sg)

			// Set current scaling group
			*scalingGroup = sg

			// Call update function
			updateCurrentGroup(doRunUpdate)

			// Wait a bit and ask for confirmation before continuing (only when more groups will follow)
			if sg < maxScale {
				fmt.Printf("Waiting %s before continuing with scaling group %v ...\n", sliceDelay.String(), sg+1)
				time.Sleep(sliceDelay)

				// Ask for confirmation to continue
				if err := confirm(fmt.Sprintf("Are you sure to continue with scaling group %v  on '%s'? Enter yes:", sg+1, stack)); err != nil {
					panic(err)
				}
			}
		}
	}
}

// detectLargestScalingGroup tries to detect the largest scaling group that still yields units.
func detectLargestScalingGroup(scalingGroup *uint, defaultScale uint, updateCurrentGroup func(runUpdate runUpdateCallback)) uint {
	// Start with 2 since we assume there is always at least 1 scaling group
	for sg := uint(2); sg <= defaultScale; sg++ {
		// Set current scaling group
		*scalingGroup = sg
		var hasUnits bool
		updateCurrentGroup(func(stack, tunnel string, unitNames, files []string, stopDelay, destroyDelay time.Duration) {
			hasUnits = len(unitNames) > 0
		})
		if !hasUnits {
			return sg - 1
		}
	}
	return defaultScale
}
