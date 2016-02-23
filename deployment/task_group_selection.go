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
	"github.com/pulcy/j2/jobs"
)

// TaskGroupSelection is a list of groups to include in the deployment.
// If empty, all groups are included.
type TaskGroupSelection []jobs.TaskGroupName

// IncludeAll returns true if the given selection is empty, false otherwise.
func (tgs TaskGroupSelection) IncludeAll() bool {
	return len(tgs) == 0
}

// Includes returns true if the given name is included in the given selection, or false otherwise.
// If the selection is empty, this will always return true.
func (tgs TaskGroupSelection) Includes(name jobs.TaskGroupName) bool {
	if len(tgs) == 0 {
		return true
	}
	for _, x := range tgs {
		if x == name {
			return true
		}
	}
	return false
}
