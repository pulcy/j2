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

	"github.com/pulcy/j2/cluster"
	"github.com/pulcy/j2/deployment"
	"github.com/pulcy/j2/extpoints"
	fg "github.com/pulcy/j2/flags"
	"github.com/pulcy/j2/jobs"
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
	deploymentDefaults(cmd.Flags(), &destroyFlags.Flags, args)
	cluster, err := loadCluster(&destroyFlags.Flags)
	if err != nil {
		Exitf("Cannot load cluster: %v\n", err)
	}
	orchestrator, err := getOrchestrator(cluster)
	if err != nil {
		Exitf("Cannot initialize orchestrator: %v\n", err)
	}
	destroyValidators(&destroyFlags.Flags, *cluster, orchestrator)

	job := jobs.Job{
		Name: jobs.JobName(destroyFlags.JobPath),
	}
	delays := deployment.DeploymentDelays{
		StopDelay:    destroyFlags.StopDelay,
		DestroyDelay: destroyFlags.DestroyDelay,
		SliceDelay:   destroyFlags.SliceDelay,
	}
	d, err := deployment.NewDeployment(orchestrator, job, *cluster,
		groups(&destroyFlags.Flags),
		deployment.ScalingGroupSelection(destroyFlags.ScalingGroup),
		destroyFlags.Force,
		destroyFlags.AutoContinue,
		globalFlags.verbose,
		delays,
		renderCtx)
	assert(err)

	assert(d.Destroy())
}

func destroyValidators(f *fg.Flags, cluster cluster.Cluster, orchestrator extpoints.Orchestrator) {
	j, err := loadJob(f, cluster, orchestrator)
	if err == nil {
		f.JobPath = j.Name.String()
	}
	jn := jobs.JobName(f.JobPath)
	if err := jn.Validate(); err != nil {
		Exitf("--job invalid: %v\n", err)
	}
}
