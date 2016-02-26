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

package units

type UnitChain []*Unit

// Link adds Require & After attributes on the units in the chain
// in order to enfore the correct chain.
func (chain UnitChain) Link() {
	for i, u := range chain {
		if i+1 < len(chain) {
			// Forward requirement
			u.ExecOptions.Require(chain[i+1].FullName)
		}
		if i > 0 {
			// Backward requirement and after
			u.ExecOptions.Require(chain[i-1].FullName)
			u.ExecOptions.After(chain[i-1].FullName)
			// machine-of previous unit in chain
			u.FleetOptions.MachineOf(chain[i-1].FullName)
		} else if len(chain) > 0 {
			// require last unit in chain
			u.ExecOptions.Require(chain[len(chain)-1].FullName)
		}
	}
}
