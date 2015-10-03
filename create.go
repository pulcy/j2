package main

import (
	"fmt"

	fg "arvika.pulcy.com/pulcy/deployit/flags"
	"arvika.pulcy.com/pulcy/deployit/fleet"

	"github.com/juju/errgo"
	"github.com/spf13/cobra"
)

var (
	createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create services on a stack.",
		Long:  "Create services on a stack.",
		Run:   createRun,
	}
)

func createUnits(tunnel string, files []string) error {
	f := fleet.NewTunnel(tunnel)

	out, err := f.Start(files...)
	if err != nil {
		return errgo.Mask(err)
	}

	fmt.Println(out)
	return nil
}

// deploymentCommandCreateRun is dynamically used for each command in
// deploymentCommands, to deploy our sets. E.g. `create base`.
func deploymentCommandCreateRun(cmd *cobra.Command, args []string) {
	dc, ok := deploymentCommands[cmd.Name()]
	if !ok {
		Exitf("unknown command: " + cmd.Name())
	}

	dc.Defaults(deploymentFlags)
	globalDefaults(deploymentFlags)

	dc.Validate(deploymentFlags)
	createValidators(deploymentFlags)
	globalValidators(deploymentFlags)

	generator := dc.ServiceGroup(deploymentFlags).Generate(deploymentFlags.Service, deploymentFlags.ScalingGroup)
	assert(generator.WriteTmpFiles())

	files := generator.FileNames()

	if deploymentFlags.DryRun {
		confirm(fmt.Sprintf("remove tmp files from %s ?", generator.TmpDir()))
	} else {
		assert(createUnits(deploymentFlags.Tunnel, files))
	}

	assert(generator.RemoveTmpFiles())
}

func createValidators(f *fg.Flags) {
}

func createRun(cmd *cobra.Command, args []string) {
	cmd.Help()
}
