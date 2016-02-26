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

import (
	"fmt"
	"strings"
)

type fleetOptions struct {
	IsGlobal      bool
	ConflictsWith []string
	machineOf     []string
	MachineID     string
	Metadata      []string
}

func NewFleetOptions() *fleetOptions {
	return &fleetOptions{
		IsGlobal:      false,
		ConflictsWith: []string{},
		Metadata:      []string{},
	}
}

func (f *fleetOptions) Conflicts(conflicts string) {
	f.ConflictsWith = append(f.ConflictsWith, conflicts)
}

// MachineMetadata adds a new metadata rule to for a service. Since one rule can define
// exclusive matching condition metadataValues is a variadic argument. See
// https://coreos.com/docs/launching-containers/launching/fleet-unit-files/#user-defined-requirements
// for more information on fleet's behaviour.
func (f *fleetOptions) MachineMetadata(metadataValues ...string) {
	if len(metadataValues) > 0 {
		// Strings have to be concatenated as double quote encapsulated strings for fleet
		metadataRule := fmt.Sprintf("\"%s\"", strings.Join(metadataValues, "\" \""))
		f.Metadata = append(f.Metadata, metadataRule)
	}
}

func (f *fleetOptions) MachineOf(otherUnit string) {
	f.machineOf = append(f.machineOf, otherUnit)
}

func (f *fleetOptions) Global() {
	f.IsGlobal = true
}
