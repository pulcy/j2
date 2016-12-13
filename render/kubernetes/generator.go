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

package kubernetes

import (
	"encoding/json"

	"github.com/pulcy/j2/cluster"
	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/render"
)

type k8sRenderer struct {
	job jobs.Job
	render.RenderConfig
}

type RenderContext interface {
	ProjectName() string
	ProjectVersion() string
	ProjectBuild() string
}

func NewGenerator(job jobs.Job, config render.RenderConfig) render.Renderer {
	return &k8sRenderer{
		job:          job,
		RenderConfig: config,
	}
}

type generatorContext struct {
	ScalingGroup  uint
	InstanceCount int
	Cluster       cluster.Cluster
}

func (g *k8sRenderer) GenerateUnits(ctx render.RenderContext, instanceCount int) ([]render.UnitData, error) {
	units := []render.UnitData{}
	maxCount := g.job.MaxCount()
	for scalingGroup := uint(1); scalingGroup <= maxCount; scalingGroup++ {
		if g.CurrentScalingGroup != 0 && g.CurrentScalingGroup != scalingGroup {
			continue
		}
		for _, tg := range g.job.Groups {
			if !g.include(tg.Name) {
				// We do not want this task group now
				continue
			}
			genCtx := generatorContext{
				ScalingGroup:  scalingGroup,
				InstanceCount: instanceCount,
				Cluster:       g.Cluster,
			}
			if deployments, err := createDeployments(tg, genCtx); err != nil {
				return nil, maskAny(err)
			} else {
				for _, res := range deployments {
					if data, err := renderResource(res.Name, res); err != nil {
						return nil, maskAny(err)
					} else {
						units = append(units, data)
					}
				}
			}
			if daemonSets, err := createDaemonSets(tg, genCtx); err != nil {
				return nil, maskAny(err)
			} else {
				for _, res := range daemonSets {
					if data, err := renderResource(res.Name, res); err != nil {
						return nil, maskAny(err)
					} else {
						units = append(units, data)
					}
				}
			}
		}
	}
	return units, nil
}

// Should the group with given name be generated?
func (g *k8sRenderer) include(groupName jobs.TaskGroupName) bool {
	if len(g.Groups) == 0 {
		// include all
		return true
	}
	for _, n := range g.Groups {
		if n == groupName {
			return true
		}
	}
	return false
}

func renderResource(unitName string, resource interface{}) (render.UnitData, error) {
	raw, err := json.Marshal(resource)
	if err != nil {
		return nil, maskAny(err)
	}
	return newUnitData(unitName+".json", string(raw)), nil
}
