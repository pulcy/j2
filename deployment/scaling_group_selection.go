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

// ScalingGroupSelection is a number indicating which scaling group (1...) to include in the deployment
// operation. Value 0 means include all.
type ScalingGroupSelection uint

// IncludeAll returns true if the given selection is empty, false otherwise.
func (sgs ScalingGroupSelection) IncludeAll() bool {
	return sgs == 0
}

// Includes returns true if the given scaling is included in the given selection, or false otherwise.
// If the selection is empty, this will always return true.
func (sgs ScalingGroupSelection) Includes(scalingGroup uint) bool {
	return (sgs == 0) || (uint(sgs) == scalingGroup)
}
