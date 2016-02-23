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
	"fmt"
	"strings"
	"time"

	"github.com/pulcy/j2/cluster"
	fg "github.com/pulcy/j2/flags"
	"github.com/pulcy/j2/fleet"
	"github.com/pulcy/j2/jobs"

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
	deploymentDefaults(cmd.Flags(), &destroyFlags.Flags, args)
	cluster, err := loadCluster(&destroyFlags.Flags)
	if err != nil {
		Exitf("Cannot load cluster: %v\n", err)
	}
	destroyValidators(&destroyFlags.Flags, *cluster)

	f := fleet.NewTunnel(cluster.Tunnel)
	list, err := f.List()
	assert(err)

	unitNames := selectUnits(list, &destroyFlags.Flags)
	if len(unitNames) == 0 {
		fmt.Printf("No units on the cluster match the given arguments\n")
	} else {
		assert(confirmDestroy(destroyFlags.Force, cluster.Stack, unitNames))
		assert(destroyUnits(cluster.Stack, f, unitNames, destroyFlags.StopDelay))
	}
}

func destroyValidators(f *fg.Flags, cluster cluster.Cluster) {
	j, err := loadJob(f, cluster)
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
	if !force {
		if err := confirm(fmt.Sprintf("You are about to destroy:\n- %s\n\nAre you sure you want to destroy %d units on stack '%s'?\nEnter yes:", strings.Join(units, "\n- "), len(units), stack)); err != nil {
			return errgo.Mask(err)
		}
	}
	fmt.Println()

	return nil
}

func destroyUnits(stack string, f *fleet.FleetTunnel, units []string, stopDelay time.Duration) error {
	if len(units) == 0 {
		return errgo.Newf("No units on cluster: %s", stack)
	}

	var out string
	out, err := f.Stop(units...)
	if err != nil {
		fmt.Printf("Warning: stop failed.\n%s\n", err.Error())
	}

	fmt.Println(out)

	fmt.Printf("Waiting for %s...\n", stopDelay)
	time.Sleep(stopDelay)

	out, err = f.Destroy(units...)
	if err != nil {
		return errgo.Mask(err)
	}

	fmt.Println(out)

	return nil
}
