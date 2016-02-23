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
	"os"

	"github.com/pulcy/j2/jobs"
)

type scalingGroupUnits struct {
	scalingGroup uint
	unitNames    []string
	fileNames    []string
}

// generateScalingGroupUnits generates the unit files for the given scaling group and returns
// their names and file names.
func (d *Deployment) generateScalingGroupUnits(scalingGroup uint) (scalingGroupUnits, error) {
	generator := d.job.Generate(jobs.GeneratorConfig{
		Groups:              d.groupSelection,
		CurrentScalingGroup: scalingGroup,
		DockerOptions:       d.cluster.DockerOptions,
		FleetOptions:        d.cluster.FleetOptions,
	})

	if err := generator.WriteTmpFiles(d.renderContext, d.images, d.cluster.InstanceCount); err != nil {
		return scalingGroupUnits{}, maskAny(err)
	}

	return scalingGroupUnits{
		scalingGroup: scalingGroup,
		unitNames:    generator.UnitNames(),
		fileNames:    generator.FileNames(),
	}, nil
}

// cleanup removes all generated files
func (sgu *scalingGroupUnits) cleanup() error {
	for _, fileName := range sgu.fileNames {
		if err := os.Remove(fileName); err != nil && !os.IsNotExist(err) {
			return maskAny(err)
		}
	}
	return nil
}
