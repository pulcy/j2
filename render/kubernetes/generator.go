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
	"strings"

	"github.com/pulcy/j2/cluster"
	"github.com/pulcy/j2/jobs"
	k8s "github.com/pulcy/j2/pkg/kubernetes"
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
	Cluster   cluster.Cluster
	Namespace string
}

func (g *k8sRenderer) GenerateUnits(ctx render.RenderContext, instanceCount int) ([]render.UnitData, error) {
	units := []render.UnitData{}
	for _, tg := range g.job.Groups {
		if !g.include(tg.Name) {
			// We do not want this task group now
			continue
		}
		genCtx := generatorContext{
			Cluster:   g.Cluster,
			Namespace: strings.Replace(g.job.Name.String(), "_", "-", -1),
		}
		pods, err := groupTaskIntoPods(tg)
		if err != nil {
			return nil, maskAny(err)
		}
		for _, p := range pods {
			if deployments, err := createDeployments(tg, p, genCtx); err != nil {
				return nil, maskAny(err)
			} else {
				for _, res := range deployments {
					units = append(units, &k8s.Deployment{Deployment: res})
				}
			}
			if daemonSets, err := createDaemonSets(tg, p, genCtx); err != nil {
				return nil, maskAny(err)
			} else {
				for _, res := range daemonSets {
					units = append(units, &k8s.DaemonSet{DaemonSet: res})
				}
			}
			if services, err := createServices(tg, p, genCtx); err != nil {
				return nil, maskAny(err)
			} else {
				for _, res := range services {
					units = append(units, &k8s.Service{Service: res})
				}
			}
			if ingresses, err := createIngresses(tg, p, genCtx); err != nil {
				return nil, maskAny(err)
			} else {
				for _, res := range ingresses {
					units = append(units, &k8s.Ingress{Ingress: res})
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
