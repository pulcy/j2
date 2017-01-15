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

package render

import "github.com/pulcy/j2/jobs"

type Renderer interface {
	jobs.Renderer
	GenerateUnits(job jobs.Job, ctx RenderContext, config RenderConfig, instanceCount int) ([]UnitData, error)
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

type UnitData interface {
	Name() string
	Content() string
}
