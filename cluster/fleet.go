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

package cluster

// FleetOptions contains options used to generate fleet units for the jobs that run on this cluster.
type FleetOptions struct {
	// A list of unit names to add to the `After` setting of all generated units
	After []string
	// A list of unit names to add to the `Wants` setting of all generated units
	Wants []string
	// A list of unit names to add to the `Requires` setting of all generated units
	Requires []string

	GlobalInstanceConstraints []string
}

// validate checks the values in the given cluster
func (o FleetOptions) validate() error {
	return nil
}

func (o *FleetOptions) setDefaults() {
	if len(o.GlobalInstanceConstraints) == 0 {
		o.GlobalInstanceConstraints = []string{
			"odd=true",
			"even=true",
		}
	}
}
