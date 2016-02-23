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

package deployment

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/juju/errgo"

	"github.com/pulcy/j2/cluster"
	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/units"
)

type DeploymentDelays struct {
	StopDelay    time.Duration
	DestroyDelay time.Duration
	SliceDelay   time.Duration
}

type Deployment struct {
	job                   jobs.Job
	cluster               cluster.Cluster
	groupSelection        TaskGroupSelection
	scalingGroupSelection ScalingGroupSelection
	force                 bool
	DeploymentDelays
	renderContext units.RenderContext
	images        jobs.Images

	scalingGroups []scalingGroupUnits
}

type DeploymentDependencies struct {
	Confirm  func(string) error
	Verbosef func(format string, args ...interface{})
}

// NewDeployment creates a new Deployment instances and generates all unit files for the given job.
func NewDeployment(job jobs.Job, cluster cluster.Cluster, groupSelection TaskGroupSelection,
	scalingGroupSelection ScalingGroupSelection, force bool, delays DeploymentDelays,
	renderContext units.RenderContext, images jobs.Images) *Deployment {
	return &Deployment{
		job:                   job,
		cluster:               cluster,
		groupSelection:        groupSelection,
		scalingGroupSelection: scalingGroupSelection,
		force:            force,
		DeploymentDelays: delays,
		renderContext:    renderContext,
		images:           images,
	}
}

// DryRun creates all unit files it will deploy during a normal `Run` and present them to the user.
func (d *Deployment) DryRun(deps DeploymentDependencies) error {
	if err := d.generateScalingGroups(); err != nil {
		return maskAny(err)
	}
	var dir string
	units := []string{}
	for _, sgu := range d.scalingGroups {
		if dir == "" && len(sgu.fileNames) > 0 {
			dir = filepath.Dir(sgu.fileNames[0])
		}
		units = append(units, sgu.unitNames...)
	}
	sort.Strings(units)
	fmt.Printf("The following units will be deployed.\n\n%s\n\nYou can review them in %s.\n",
		strings.Join(units, "\n"),
		dir,
	)
	if err := deps.Confirm(fmt.Sprintf("Remove review files from %s ?", dir)); err != nil {
		return maskAny(err)
	}

	if err := d.cleanup(); err != nil {
		return maskAny(errgo.Notef(err, "Failed to cleanup files"))
	}

	return nil
}

// generateScalingGroups generates the unit files for all scaling groups included in the selection.
// After this, a call to cleanup is needed.
func (d *Deployment) generateScalingGroups() error {
	maxCount := d.job.MaxCount()
	for scalingGroup := uint(1); scalingGroup <= maxCount; scalingGroup++ {
		if !d.scalingGroupSelection.Includes(scalingGroup) {
			continue
		}
		sgu, err := d.generateScalingGroupUnits(scalingGroup)
		if err != nil {
			return maskAny(err)
		}
		d.scalingGroups = append(d.scalingGroups, sgu)
	}
	return nil
}

// cleanup removes all temp files.
func (d *Deployment) cleanup() error {
	for _, sgu := range d.scalingGroups {
		if err := sgu.cleanup(); err != nil {
			return maskAny(err)
		}
	}
	return nil
}
