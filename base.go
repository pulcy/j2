package main

import (
	"arvika.pulcy.com/iggi/deployit/base"
	fg "arvika.pulcy.com/iggi/deployit/flags"

	"github.com/spf13/pflag"
)

func init() {
	deploymentCommands["base"] = DeploymentCommand{
		Short: "Release base components of a cluster.",

		Long: []string{
			"All clusters are deployed with the same set of components.",
			"This command helps releasing and configuring the components on a cluster.",
		},

		Flags: func(fs *pflag.FlagSet, f *fg.Flags) {
			// Image versions
			fs.StringVar(&f.LoadBalancerVersion, "load-balancer-version", defaultLoadBalancerVersion, "Version of load-balancer")
		},

		Defaults: func(f *fg.Flags) {
		},

		Validate: func(f *fg.Flags) {
		},

		ServiceGroup: base.NewBase,
	}
}
