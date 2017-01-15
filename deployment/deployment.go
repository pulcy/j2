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
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/juju/errgo"

	"github.com/pulcy/j2/cluster"
	"github.com/pulcy/j2/extpoints"
	"github.com/pulcy/j2/jobs"
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
	verbose               bool
	force                 bool
	autoContinue          bool
	DeploymentDelays
	renderContext RenderContext
	orchestrator  extpoints.Orchestrator

	scalingGroups []scalingGroupUnits
}

type RenderContext interface {
	ProjectName() string
	ProjectVersion() string
	ProjectBuild() string

	ImageVaultMonkey() string // Docker image name of vault-monkey
	ImageWormhole() string    // Docker image name of wormhole
	ImageAlpine() string      // Docker image name of alpine linux
	ImageCephVolume() string  // Docker image name of ceph-volume
}

// NewDeployment creates a new Deployment instances and generates all unit files for the given job.
func NewDeployment(orchestrator extpoints.Orchestrator, job jobs.Job, cluster cluster.Cluster, groupSelection TaskGroupSelection,
	scalingGroupSelection ScalingGroupSelection, force, autoContinue, verbose bool, delays DeploymentDelays,
	renderContext RenderContext) (*Deployment, error) {
	return &Deployment{
		job:                   job,
		cluster:               cluster,
		groupSelection:        groupSelection,
		scalingGroupSelection: scalingGroupSelection,
		force:            force,
		autoContinue:     autoContinue,
		verbose:          verbose,
		DeploymentDelays: delays,
		renderContext:    renderContext,
		orchestrator:     orchestrator,
	}, nil
}

// DryRun creates all unit files it will deploy during a normal `Run` and present them to the user.
func (d *Deployment) DryRun() error {
	ui := newStateUI(d.verbose)
	defer ui.Close()

	if err := d.generateScalingGroups(); err != nil {
		return maskAny(err)
	}
	dir, err := ioutil.TempDir("", "j2")
	if err != nil {
		return maskAny(err)
	}
	unitNames := []string{}
	for _, sgu := range d.scalingGroups {
		for _, u := range sgu.units {
			unitPath := filepath.Join(dir, u.Name())
			if err := ioutil.WriteFile(unitPath, []byte(u.Content()), 0644); err != nil {
				return maskAny(err)
			}
			unitNames = append(unitNames, u.Name())
		}
	}
	sort.Strings(unitNames)
	ui.HeaderSink <- fmt.Sprintf("The following units will be deployed.\n\n%s\n\nYou can review them in %s.\n",
		strings.Join(unitNames, "\n"),
		dir,
	)

	if err := ui.Confirm(fmt.Sprintf("Remove review files from %s ?", dir)); err != nil {
		return maskAny(err)
	}

	if err := os.RemoveAll(dir); err != nil {
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
