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

type execOptions struct {
	// Service
	IsOneshot               bool
	IsForking               bool
	RemainsAfterExit        bool
	Restart                 string
	RestartSec              uint8
	StartLimitInterval      string
	StartLimitBurst         uint8
	TimeoutStartSec         uint8
	ContainerTimeoutStopSec uint8
	EnvironmentFiles        []string
	Environment             map[string]string
	ExecStartPre            []string
	ExecStart               string
	ExecStartPost           []string
	ExecStop                []string
	ExecStopPost            []string
	BindsTos                []string
	Wants                   string
	after                   []string
	Requires                []string

	// Timer
	OnCalendar string
	Unit       string
}

func NewExecOptions(start ...string) *execOptions {
	return &execOptions{
		IsOneshot:               false,
		IsForking:               false,
		RemainsAfterExit:        false,
		Restart:                 "on-failure",
		RestartSec:              1,
		StartLimitInterval:      "60s",
		StartLimitBurst:         3,
		TimeoutStartSec:         0,
		ContainerTimeoutStopSec: 10,
		EnvironmentFiles:        []string{"/etc/environment"},
		Environment:             make(map[string]string),
		ExecStart:               strings.Join(start, " "),
	}
}

func (e *execOptions) Oneshot() {
	e.IsOneshot = true
}

func (e *execOptions) Forking() {
	e.IsForking = true
}

func (e *execOptions) RemainAfterExit() {
	e.RemainsAfterExit = true
}

func (e *execOptions) SetRestartSec(seconds uint8) {
	e.RestartSec = seconds
}

func (e *execOptions) After(after ...string) {
	e.after = append(e.after, after...)
}

func (e *execOptions) Require(require ...string) {
	e.Requires = append(e.Requires, require...)
}

type fleetOptions struct {
	IsGlobal      bool
	ConflictsWith []string
	MachineOf     string
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

func (f *fleetOptions) Global() {
	f.IsGlobal = true
}
