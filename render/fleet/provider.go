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

package fleet

import "github.com/pulcy/j2/render"
import "github.com/pulcy/j2/cluster"

type fleetProvider struct {
}

// NewRenderProvider creates a new render provider that will render fleet units.
func NewRenderProvider() render.RenderProvider {
	return &fleetProvider{}
}

func (p *fleetProvider) CreateRenderer(cluster.Cluster) render.Renderer {
	return newGenerator()
}
