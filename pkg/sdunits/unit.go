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

package sdunits

type Unit struct {
	Name            string // e.g. "foo"
	FullName        string // e.g. "foo@1.service"
	Type            string "service|socket|timer"
	Description     string
	Scalable_       bool
	ScalingGroup    uint
	ExecOptions     *execOptions
	FleetOptions    *fleetOptions
	projectSettings map[string]string
}

// ProjectSetting as a key-value to the `X-<project>` section
func (unit *Unit) ProjectSetting(key, value string) {
	if unit.projectSettings == nil {
		unit.projectSettings = make(map[string]string)
	}
	unit.projectSettings[key] = value
}

// GroupUnitsOnMachine modifies given list of units such
// that all units are forced on the same machine (of the first unit)
func GroupUnitsOnMachine(units ...*Unit) {
	for i, u := range units {
		if i == 0 {
			continue
		}
		u.FleetOptions.MachineOf(units[0].FullName)
	}
}
