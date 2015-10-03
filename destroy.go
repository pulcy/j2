package main

import (
	"fmt"
	"time"

	"arvika.pulcy.com/pulcy/deployit/fleet"

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
)

func destroyUnits(stack, tunnel string, list []string) error {
	if len(list) == 0 {
		return errgo.Newf("No units on cluster: %s", stack)
	}

	f := fleet.NewTunnel(tunnel)

	var out string
	out, err := f.Stop(list...)
	if err != nil {
		return errgo.Mask(err)
	}

	fmt.Println(out)

	fmt.Println("Waiting for 15 seconds...")
	time.Sleep(15 * time.Second)

	out, err = f.Destroy(list...)
	if err != nil {
		return errgo.Mask(err)
	}

	fmt.Println(out)

	return nil
}

// deploymentCommandUpdateRun is dynamically used for each command in
// deploymentCommands, to deploy our sets. E.g. `destroy base`.
func deploymentCommandDestroyRun(cmd *cobra.Command, args []string) {
	dc, ok := deploymentCommands[cmd.Name()]
	if !ok {
		Exitf("unknown command: " + cmd.Name())
	}

	dc.Defaults(deploymentFlags)
	globalDefaults(deploymentFlags)

	globalValidators(deploymentFlags)

	generator := dc.ServiceGroup(deploymentFlags).Generate(deploymentFlags.Service, deploymentFlags.ScalingGroup)
	assert(generator.WriteTmpFiles())

	unitNames := generator.UnitNames()

	assert(confirmDestroy(deploymentFlags.Force, deploymentFlags.Service, deploymentFlags.Stack, unitNames))
	assert(destroyUnits(deploymentFlags.Stack, deploymentFlags.Tunnel, unitNames))

	assert(generator.RemoveTmpFiles())
}

func confirmDestroy(force bool, group, stack string, list []string) error {
	for _, unit := range list {
		fmt.Println(unit)
	}
	fmt.Println()

	if !force {
		if err := confirm(fmt.Sprintf("Are you sure you want to destroy %s units on '%s'? Enter yes:", group, stack)); err != nil {
			return errgo.Mask(err)
		}
	}
	fmt.Println()

	return nil
}

func destroyRun(cmd *cobra.Command, args []string) {
	if deploymentFlags.Stack == "" {
		Exitf("--stack missing")
	}

	if deploymentFlags.Tunnel == "" {
		deploymentFlags.Tunnel = fmt.Sprintf("%s.iggi.xyz", deploymentFlags.Stack)
	}

	f := fleet.NewTunnel(deploymentFlags.Tunnel)

	list, err := f.List()
	assert(err)

	assert(confirmDestroy(deploymentFlags.Force, deploymentFlags.Service, deploymentFlags.Stack, list))
	assert(destroyUnits(deploymentFlags.Stack, deploymentFlags.Tunnel, list))
}
