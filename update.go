package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var (
	updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Update services on a stack.",
		Long:  "Update services on a stack.",
		Run:   updateRun,
	}
)

func doRunUpdate(stack, tunnel string, unitNames, files []string) {
	if len(unitNames) != len(files) {
		panic("Internal update error")
	}

	assert(destroyUnits(stack, tunnel, unitNames))

	fmt.Println("Waiting for 15 seconds ...")
	time.Sleep(15 * time.Second)

	assert(createUnits(tunnel, files))
}

type runUpdateCallback func(stack, tunnel string, unitNames, files []string)

// updateScalingGroups calls a given update function for each scaling group, such that they all update in succession.
// confirmation is asked before updating more than 1 scaling group
func updateScalingGroups(scalingGroup *uint8, scale uint8, stack string, updateCurrentGroup func(runUpdate runUpdateCallback)) {
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
		for sg := uint8(1); sg <= maxScale; sg++ {
			// Tell what we're going to do
			fmt.Printf("Updating scaling group %v ...\n", sg)

			// Set current scaling group
			*scalingGroup = sg

			// Call update function
			updateCurrentGroup(doRunUpdate)

			// Wait a bit and ask for confirmation before continuing (only when more groups will follow)
			if sg < maxScale {
				fmt.Printf("Waiting %s before continuing with scaling group %v ...\n", globalFlags.sleep.String(), sg+1)
				time.Sleep(globalFlags.sleep)

				// Ask for confirmation to continue
				if err := confirm(fmt.Sprintf("Are you sure to continue with scaling group %v  on '%s'? Enter yes:", sg+1, stack)); err != nil {
					panic(err)
				}
			}
		}
	}
}

// detectLargestScalingGroup tries to detect the largest scaling group that still yields units.
func detectLargestScalingGroup(scalingGroup *uint8, defaultScale uint8, updateCurrentGroup func(runUpdate runUpdateCallback)) uint8 {
	// Start with 2 since we assume there is always at least 1 scaling group
	for sg := uint8(2); sg <= defaultScale; sg++ {
		// Set current scaling group
		*scalingGroup = sg
		var hasUnits bool
		updateCurrentGroup(func(stack, tunnel string, unitNames, files []string) {
			hasUnits = len(unitNames) > 0
		})
		if !hasUnits {
			return sg - 1
		}
	}
	return defaultScale
}

// deploymentCommandUpdateRun is dynamically used for each command in
// deploymentCommands, to deploy our sets. E.g. `update base`.
func deploymentCommandUpdateRun(cmd *cobra.Command, args []string) {
	dc, ok := deploymentCommands[cmd.Name()]
	if !ok {
		Exitf("unknown command: " + cmd.Name())
	}

	dc.Defaults(deploymentFlags)
	globalDefaults(deploymentFlags)

	dc.Validate(deploymentFlags)
	createValidators(deploymentFlags)
	globalValidators(deploymentFlags)

	scale := uint8(deploymentFlags.DefaultScale)

	location := deploymentFlags.Stack
	updateScalingGroups(&deploymentFlags.ScalingGroup, scale, location, func(runUpdate runUpdateCallback) {
		generator := dc.ServiceGroup(deploymentFlags).Generate(deploymentFlags.Service, deploymentFlags.ScalingGroup)

		assert(generator.WriteTmpFiles())

		unitNames := generator.UnitNames()
		fileNames := generator.FileNames()

		runUpdate(deploymentFlags.Stack, deploymentFlags.Tunnel, unitNames, fileNames)
		assert(generator.RemoveTmpFiles())
	})
}

func updateRun(cmd *cobra.Command, args []string) {
	cmd.Help()
}
