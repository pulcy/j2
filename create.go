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
		Short: "Create a job on a stack.",
		Long:  "Create a job on a stack.",
		Run:   createRun,
	}
	createFlags struct {
		fg.Flags
	}
)

func init() {
	initDeploymentFlags(createCmd.Flags(), &createFlags.Flags)
}

func createRun(cmd *cobra.Command, args []string) {
	deploymentDefaults(&createFlags.Flags, args)
	createValidators(&createFlags.Flags)
	deploymentValidators(&createFlags.Flags)

	job, err := loadJob(&createFlags.Flags)
	if err != nil {
		Exitf("Cannot load job: %v\n", err)
	}
	groups := groups(&createFlags.Flags)
	generator := job.Generate(groups, createFlags.ScalingGroup)
	assert(generator.WriteTmpFiles())

	files := generator.FileNames()

	if createFlags.DryRun {
		confirm(fmt.Sprintf("remove tmp files from %s ?", generator.TmpDir()))
	} else {
		assert(createUnits(createFlags.Tunnel, files))
	}

	assert(generator.RemoveTmpFiles())
}

func createValidators(f *fg.Flags) {
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
