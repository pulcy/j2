package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/juju/errgo"
	"github.com/spf13/cobra"

	fg "arvika.pulcy.com/pulcy/deployit/flags"
	"arvika.pulcy.com/pulcy/deployit/fleet"
	"arvika.pulcy.com/pulcy/deployit/units"
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
	deploymentValidators(&runFlags.Flags)

	job, err := loadJob(&runFlags.Flags)
	if err != nil {
		Exitf("Cannot load job: %v\n", err)
	}
	groups := groups(&runFlags.Flags)
	generator := job.Generate(groups, runFlags.ScalingGroup)
	assert(generator.WriteTmpFiles(ctx, runFlags.InstanceCount))

	if runFlags.DryRun {
		confirm(fmt.Sprintf("remove tmp files from %s ?", generator.TmpDir()))
	} else {
		location := runFlags.Stack
		count := job.MaxCount()
		updateScalingGroups(&runFlags.ScalingGroup, count, location, func(runUpdate runUpdateCallback) {
			generator := job.Generate(groups, runFlags.ScalingGroup)

			assert(generator.WriteTmpFiles(ctx, runFlags.InstanceCount))

			unitNames := generator.UnitNames()
			fileNames := generator.FileNames()

			runUpdate(runFlags.Stack, runFlags.Tunnel, unitNames, fileNames, runFlags.StopDelay, runFlags.DestroyDelay)
			assert(generator.RemoveTmpFiles())
		}, runFlags.SliceDelay, runFlags.Force)
	}
}

func runValidators(f *fg.Flags) {
}

func doRunUpdate(stack, tunnel string, unitNames, files []string, stopDelay, destroyDelay time.Duration) {
	if len(unitNames) != len(files) {
		panic("Internal update error")
	}

	f := fleet.NewTunnel(tunnel)
	loadedUnitNames, err := selectLoadedUnits(unitNames, f)
	assert(err)
	if len(loadedUnitNames) > 0 {
		assert(destroyUnits(stack, f, loadedUnitNames, stopDelay))
		fmt.Printf("Waiting for %s...\n", destroyDelay)
		time.Sleep(destroyDelay)
	}

	assert(createUnits(tunnel, files))
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

func contains(list []string, value string) bool {
	for _, x := range list {
		if x == value {
			return true
		}
	}
	return false
}

type runUpdateCallback func(stack, tunnel string, unitNames, files []string, stopDelay, destroyDelay time.Duration)

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
	updateCurrentGroup(func(stack, tunnel string, unitNames, files []string, stopDelay, destroyDelay time.Duration) {
		names = unitNames
	})
	for sg := uint(2); sg <= defaultScale; sg++ {
		// Set current scaling group
		*scalingGroup = sg
		var hasUnits bool
		updateCurrentGroup(func(stack, tunnel string, unitNames, files []string, stopDelay, destroyDelay time.Duration) {
			hasUnits = len(unitNames) > 0
		})
		if !hasUnits {
			return sg - 1, names
		}
	}
	return defaultScale, names
}

func createUnits(tunnel string, files []string) error {
	f := fleet.NewTunnel(tunnel)

	out, err := f.Start(files...)
	if err != nil {
		return errgo.Mask(err)
	}

	fmt.Println(out)
	return nil
}
