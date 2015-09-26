package main

import (
	"fmt"
	"strings"

	fg "arvika.pulcy.com/iggi/deployit/flags"
	"arvika.pulcy.com/iggi/deployit/services"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type DeploymentCommand struct {
	Short        string
	Long         []string
	Flags        func(fs *pflag.FlagSet, f *fg.Flags)
	Defaults     func(f *fg.Flags)
	Validate     func(f *fg.Flags)
	ServiceGroup func(f *fg.Flags) *services.ServiceGroup
}

var (
	deploymentCommands = map[string]DeploymentCommand{}
	deploymentFlags    = &fg.Flags{}
)

func initGlobalFlags(fs *pflag.FlagSet, f *fg.Flags) {
	fs.StringVar(&f.Stack, "stack", defaultStack, "stack name of the cluster")
	fs.StringVar(&f.Tunnel, "tunnel", defaultTunnel, "SSH endpoint to tunnel through with fleet")
	fs.StringVar(&f.Service, "service", defaultService, "target service to deploy")
	fs.BoolVar(&f.Force, "force", defaultForce, "wheather to confirm destroy or not")
	fs.BoolVar(&f.DryRun, "dry-run", defaultDryRun, "wheather to schedule units or not")
	fs.Uint8Var(&f.ScalingGroup, "scaling-group", defaultScalingGroup, "scaling group to deploy")
	fs.Uint8Var(&f.DefaultScale, "default-scale", defaultDefaultScale, "total number of services")
	fs.StringVar(&f.PrivateRegistry, "private-registry", defaultPrivateRegistry, "private registry for the docker images")
	fs.StringVar(&f.LogLevel, "log-level", defaultLogLevel, "log-level for our services")
}

func createDeploymentCommands() {
	// register all create deployment commands
	for cmdName, dc := range deploymentCommands {
		// create
		subCreateCmd := &cobra.Command{
			Use:   cmdName,
			Short: dc.Short,
			Long:  strings.Join(dc.Long, " "),
			Run:   deploymentCommandCreateRun,
		}

		initGlobalFlags(subCreateCmd.Flags(), deploymentFlags)
		dc.Flags(subCreateCmd.Flags(), deploymentFlags)

		createCmd.AddCommand(subCreateCmd)

		// destroy
		subDestroyCmd := &cobra.Command{
			Use:   cmdName,
			Short: dc.Short,
			Long:  strings.Join(dc.Long, " "),
			Run:   deploymentCommandDestroyRun,
		}

		initGlobalFlags(subDestroyCmd.Flags(), deploymentFlags)
		dc.Flags(subDestroyCmd.Flags(), deploymentFlags)

		destroyCmd.AddCommand(subDestroyCmd)

		// update
		subUpdateCmd := &cobra.Command{
			Use:   cmdName,
			Short: dc.Short,
			Long:  strings.Join(dc.Long, " "),
			Run:   deploymentCommandUpdateRun,
		}

		initGlobalFlags(subUpdateCmd.Flags(), deploymentFlags)
		dc.Flags(subUpdateCmd.Flags(), deploymentFlags)

		updateCmd.AddCommand(subUpdateCmd)
	}
}

func globalDefaults(f *fg.Flags) {
	if f.Tunnel == "" {
		f.Tunnel = fmt.Sprintf("%s.iggi.xyz", f.Stack)
	}

	if f.LogLevel == "" {
		f.LogLevel = "debug"
	}
}

func globalValidators(f *fg.Flags) {
	if f.Stack == "" || f.Tunnel == "" {
		Exitf("--stack or --tunnel missing")
	}
}
