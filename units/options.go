package units

import (
	"fmt"
	"strings"
)

type execOptions struct {
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
	ExecStopPre             []string
	ExecStop                string
	ExecStopPost            []string
	BindsTos                []string
	Wants                   string
	After                   string
	MachineOf               string
}

func NewExecOptions(start ...string) *execOptions {
	return &execOptions{
		IsOneshot:               false,
		IsForking:               false,
		RemainsAfterExit:        false,
		Restart:                 "on-failure",
		RestartSec:              1,
		StartLimitInterval:      "300s",
		StartLimitBurst:         3,
		TimeoutStartSec:         0,
		ContainerTimeoutStopSec: 10,
		EnvironmentFiles:        []string{"/etc/environment"},
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

type fleetOptions struct {
	IsGlobal      bool
	HasConflicts  bool
	ConflictsWith []string
	SameMachine   string
	RawMetadata   bool
	Metadata      []string
}

func FleetOptions() *fleetOptions {
	return &fleetOptions{
		IsGlobal:      false,
		HasConflicts:  false,
		ConflictsWith: []string{},
		SameMachine:   "",
		RawMetadata:   false,
		Metadata:      []string{},
	}
}

func (f *fleetOptions) Conflicts(conflicts string) {
	f.HasConflicts = true
	f.ConflictsWith = append(f.ConflictsWith, conflicts)
}

// MachineMetadata adds a new metadata rule to for a service. Since one rule can define
// exclusive matching condition metadataValues is a variadic argument. See
// https://coreos.com/docs/launching-containers/launching/fleet-unit-files/#user-defined-requirements
// for more information on fleet's behaviour.
func (f *fleetOptions) MachineMetadata(metadataValues ...string) {
	// Strings have to be concacted as double quote encapsulated strings for fleet
	metadataRule := fmt.Sprintf("\"%s\"", strings.Join(metadataValues, "\" \""))
	f.Metadata = append(f.Metadata, metadataRule)
}

// SetRawMetadata ensures that metadata is just written to the templates as the
// user defines it. Otherwise general metadata will be automatically added to
// the templates, e.g. metadata for the stack name.
func (f *fleetOptions) SetRawMetadata() {
	f.RawMetadata = true
}

func (f *fleetOptions) Global() {
	f.IsGlobal = true
}
