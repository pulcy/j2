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
		Short: "Destroy services on a stack.",
		Long:  "Destroy services on a stack.",
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
	deploymentDefaults(&destroyFlags.Flags)
	destroyValidators(&destroyFlags.Flags)
	deploymentValidators(&destroyFlags.Flags)

	f := fleet.NewTunnel(destroyFlags.Tunnel)
	list, err := f.List()
	assert(err)

	groups := groups(&destroyFlags.Flags)
	unitNames := selectUnits(list, groups)

	assert(confirmDestroy(destroyFlags.Force, destroyFlags.Stack, unitNames))
	assert(destroyUnits(destroyFlags.Stack, f, unitNames))
}

func destroyValidators(f *fg.Flags) {
	jn := jobs.JobName(f.JobPath)
	if err := jn.Validate(); err != nil {
		Exitf("--job invalid: %v\n", err)
	}
}

func selectUnits(allUnitNames []string, groups []jobs.TaskGroupName) []string {
	return allUnitNames //TODO
}

func confirmDestroy(force bool, stack string, units []string) error {
	for _, unit := range units {
		fmt.Println(unit)
	}
	fmt.Println()

	if !force {
		if err := confirm(fmt.Sprintf("You are about to destroy:%s\n\nAre you sure you want to destroy %d units on '%s'? Enter yes:", strings.Join(units, "\n"), len(units), stack)); err != nil {
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
