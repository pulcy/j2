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

	"github.com/pulcy/j2/jobs/render"
)

type scalingGroupUnits struct {
	scalingGroup uint
	units        []render.UnitData
}

func (sgu scalingGroupUnits) unitNames() []string {
	var names []string
	for _, u := range sgu.units {
		names = append(names, u.Name())
	}
	return names
}

func (sgu scalingGroupUnits) get(unitName string) (render.UnitData, error) {
	for _, u := range sgu.units {
		if u.Name() == unitName {
			return u, nil
		}
	}
	return nil, maskAny(fmt.Errorf("unit '%s' not found", unitName))
}

func (sgu scalingGroupUnits) selectByNames(unitNames ...[]string) []render.UnitData {
	names := make(map[string]struct{})
	for _, list := range unitNames {
		for _, name := range list {
			names[name] = struct{}{}
		}
	}
	var result []render.UnitData
	for _, u := range sgu.units {
		if _, ok := names[u.Name()]; ok {
			result = append(result, u)
		}
	}
	return result
}

// generateScalingGroupUnits generates the unit files for the given scaling group and returns
// their names and file names.
func (d *Deployment) generateScalingGroupUnits(scalingGroup uint) (scalingGroupUnits, error) {
	generator := render.NewGenerator(d.job, render.GeneratorConfig{
		Groups:              d.groupSelection,
		CurrentScalingGroup: scalingGroup,
		DockerOptions:       d.cluster.DockerOptions,
		FleetOptions:        d.cluster.FleetOptions,
	})

	units, err := generator.GenerateUnits(d.renderContext, d.images, d.cluster.InstanceCount)
	if err != nil {
		return scalingGroupUnits{}, maskAny(err)
	}

	return scalingGroupUnits{
		scalingGroup: scalingGroup,
		units:        units,
	}, nil
}
