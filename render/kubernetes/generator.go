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
}

type RenderContext interface {
	ProjectName() string
	ProjectVersion() string
	ProjectBuild() string
}

func newGenerator() render.Renderer {
	return &k8sRenderer{}
}

type generatorContext struct {
	Cluster          cluster.Cluster
	Namespace        string
	ImageVaultMonkey string
}

func (g *k8sRenderer) NormalizeTask(t *jobs.Task) error {
	if t.Network.IsWeave() || t.Network.IsDefault() {
		t.Network = jobs.NetworkTypeDefault
	}
	return nil
}

func (g *k8sRenderer) GenerateUnits(job jobs.Job, ctx render.RenderContext, config render.RenderConfig, instanceCount int) ([]render.UnitData, error) {
	if config.CurrentScalingGroup != 1 {
		// We only generate units for the first scaling group.
		// Kubernetes will do the actual scaling.
		return nil, nil
	}
	units := []render.UnitData{}
	for _, tg := range job.Groups {
		if !include(config, tg.Name) {
			// We do not want this task group now
			continue
		}
		genCtx := generatorContext{
			Cluster:          config.Cluster,
			Namespace:        strings.Replace(job.Name.String(), "_", "-", -1),
			ImageVaultMonkey: ctx.ImageVaultMonkey(),
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
			if jobs, err := createJobs(tg, p, genCtx); err != nil {
				return nil, maskAny(err)
			} else {
				for _, res := range jobs {
					units = append(units, &k8s.Job{Job: res})
				}
			}
			if secrets, err := createSecrets(tg, p, genCtx); err != nil {
				return nil, maskAny(err)
			} else {
				for _, res := range secrets {
					units = append(units, &k8s.Secret{Secret: res})
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
func include(config render.RenderConfig, groupName jobs.TaskGroupName) bool {
	if len(config.Groups) == 0 {
		// include all
		return true
	}
	for _, n := range config.Groups {
		if n == groupName {
			return true
		}
	}
	return false
}
