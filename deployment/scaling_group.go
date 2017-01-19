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

	"github.com/pulcy/j2/render"
	"github.com/pulcy/j2/scheduler"
)

type scalingGroupUnits struct {
	scalingGroup uint
	units        []render.UnitData
}

func (sgu scalingGroupUnits) Units() []scheduler.Unit {
	var units []scheduler.Unit
	for _, u := range sgu.units {
		units = append(units, u)
	}
	return units
}

func (sgu scalingGroupUnits) get(unit scheduler.Unit) (render.UnitData, error) {
	for _, u := range sgu.units {
		if u.Name() == unit.Name() {
			return u, nil
		}
	}
	return nil, maskAny(fmt.Errorf("unit '%s' not found", unit.Name()))
}

func (sgu scalingGroupUnits) selectByNames(units ...[]scheduler.Unit) scheduler.UnitDataList {
	names := make(map[string]struct{})
	for _, list := range units {
		for _, u := range list {
			names[u.Name()] = struct{}{}
		}
	}
	var result []render.UnitData
	for _, u := range sgu.units {
		if _, ok := names[u.Name()]; ok {
			result = append(result, u)
		}
	}
	return scalingGroupUnits{scalingGroup: sgu.scalingGroup, units: result}
}

func (sgu scalingGroupUnits) Len() int {
	return len(sgu.units)
}

func (sgu scalingGroupUnits) Get(index int) scheduler.UnitData {
	return sgu.units[index]
}

// generateScalingGroupUnits generates the unit files for the given scaling group and returns
// their names and file names.
func (d *Deployment) generateScalingGroupUnits(scalingGroup uint) (scalingGroupUnits, error) {
	renderProvider, err := d.orchestrator.RenderProvider()
	if err != nil {
		return scalingGroupUnits{}, maskAny(err)
	}
	config := render.RenderConfig{
		Groups:              d.groupSelection,
		CurrentScalingGroup: scalingGroup,
		Cluster:             d.cluster,
	}
	renderer := renderProvider.CreateRenderer(d.cluster)
	units, err := renderer.GenerateUnits(d.job, d.renderContext, config, d.cluster.InstanceCount)
	if err != nil {
		return scalingGroupUnits{}, maskAny(err)
	}

	return scalingGroupUnits{
		scalingGroup: scalingGroup,
		units:        units,
	}, nil
}
