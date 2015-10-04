package main

import (
	"fmt"
	"strings"
	"time"

	fg "arvika.pulcy.com/pulcy/deployit/flags"
	"arvika.pulcy.com/pulcy/deployit/fleet"
	"arvika.pulcy.com/pulcy/deployit/jobs"

	"github.com/juju/errgo"
	"github.com/spf13/cobra"
)

var (
	destroyCmd = &cobra.Command{
		Use:   "destroy",
		Short: "Destroy a job on a stack.",
		Long:  "Destroy a job on a stack.",
		Run:   destroyRun,
	}
	destroyFlags struct {
		fg.Flags
	}
)

func init() {
	initDeploymentFlags(destroyCmd.Flags(), &destroyFlags.Flags)
}

func destroyRun(cmd *cobra.Command, args []string) {
	deploymentDefaults(&destroyFlags.Flags, args)
	destroyValidators(&destroyFlags.Flags)
	deploymentValidators(&destroyFlags.Flags)

	f := fleet.NewTunnel(destroyFlags.Tunnel)
	list, err := f.List()
	assert(err)

	unitNames := selectUnits(list, &destroyFlags.Flags)
	if len(unitNames) == 0 {
		fmt.Printf("No units on the cluster match the given arguments\n")
	} else {
		assert(confirmDestroy(destroyFlags.Force, destroyFlags.Stack, unitNames))
		assert(destroyUnits(destroyFlags.Stack, f, unitNames))
	}
}

func destroyValidators(f *fg.Flags) {
	j, err := loadJob(f)
	if err == nil {
		f.JobPath = j.Name.String()
	}
	jn := jobs.JobName(f.JobPath)
	if err := jn.Validate(); err != nil {
		Exitf("--job invalid: %v\n", err)
	}
}

func selectUnits(allUnitNames []string, f *fg.Flags) []string {
	groups := groups(f)
	var filter func(string) bool
	jobName := jobs.JobName(f.JobPath)
	if len(groups) == 0 {
		// Select everything in the job
		filter = func(unitName string) bool {
			return jobs.IsUnitForJob(unitName, jobName)
		}
	} else {
		// Select everything in one of the groups
		filter = func(unitName string) bool {
			for _, g := range groups {
				if jobs.IsUnitForTaskGroup(unitName, jobName, g) {
					return true
				}
			}
			return false
		}
	}
	list := []string{}
	for _, unitName := range allUnitNames {
		if filter(unitName) {
			list = append(list, unitName)
		}
	}
	return list
}

func confirmDestroy(force bool, stack string, units []string) error {
	for _, unit := range units {
		fmt.Println(unit)
	}
	fmt.Println()

	if !force {
		if err := confirm(fmt.Sprintf("You are about to destroy:\n%s\n\nAre you sure you want to destroy %d units on '%s'? Enter yes:", strings.Join(units, "\n"), len(units), stack)); err != nil {
			return errgo.Mask(err)
		}
	}
	fmt.Println()

	return nil
}

func destroyUnits(stack string, f *fleet.FleetTunnel, units []string) error {
	if len(units) == 0 {
		return errgo.Newf("No units on cluster: %s", stack)
	}

	var out string
	out, err := f.Stop(units...)
	if err != nil {
		return errgo.Mask(err)
	}

	fmt.Println(out)

	fmt.Println("Waiting for 15 seconds...")
	time.Sleep(15 * time.Second)

	out, err = f.Destroy(units...)
	if err != nil {
		return errgo.Mask(err)
	}

	fmt.Println(out)

	return nil
}
