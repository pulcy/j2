// Copyright (c) 2016 Pulcy.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/spf13/cobra"

	"github.com/pulcy/j2/deployment"
	fg "github.com/pulcy/j2/flags"
	"github.com/pulcy/j2/units"
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
	renderContext = units.RenderContext{
		ProjectName:    projectName,
		ProjectVersion: projectVersion,
		ProjectBuild:   projectBuild,
	}
)

func init() {
	initDeploymentFlags(runCmd.Flags(), &runFlags.Flags)
}

func runRun(cmd *cobra.Command, args []string) {
	deploymentDefaults(cmd.Flags(), &runFlags.Flags, args)
	runValidators(&runFlags.Flags)

	cluster, err := loadCluster(&runFlags.Flags)
	if err != nil {
		Exitf("Cannot load cluster: %v\n", err)
	}
	job, err := loadJob(&runFlags.Flags, *cluster)
	if err != nil {
		Exitf("Cannot load job: %v\n", err)
	}

	delays := deployment.DeploymentDelays{
		StopDelay:    runFlags.StopDelay,
		DestroyDelay: runFlags.DestroyDelay,
		SliceDelay:   runFlags.SliceDelay,
	}
	d := deployment.NewDeployment(*job, *cluster, groups(&runFlags.Flags),
		deployment.ScalingGroupSelection(runFlags.ScalingGroup), runFlags.Force, runFlags.AutoContinue, globalFlags.verbose, delays, renderContext, images)

	if runFlags.DryRun {
		assert(d.DryRun())
	} else {
		assert(d.Run())
	}
}

func runValidators(f *fg.Flags) {
}
