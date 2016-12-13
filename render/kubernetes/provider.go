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
	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/render"
)

type k8sProvider struct {
}

// NewRenderProvider creates a new render provider that will render kubernetes units.
func NewRenderProvider() render.RenderProvider {
	return &k8sProvider{}
}

func (p *k8sProvider) CreateRenderer(job jobs.Job, cfg render.RenderConfig) (render.Renderer, error) {
	return NewGenerator(job, cfg), nil
}
