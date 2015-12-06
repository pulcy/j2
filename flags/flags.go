package flags

import (
	"time"
)

type Flags struct {
	// Global flags
	Local         bool // Use local vagrant test cluster
	JobPath       string
	Groups        []string
	Stack         string
	Domain        string
	Tunnel        string
	ScalingGroup  uint
	DryRun        bool
	Force         bool
	StopDelay     time.Duration
	DestroyDelay  time.Duration
	SliceDelay    time.Duration
	InstanceCount int
	Options       Options
}
