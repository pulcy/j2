package flags

import (
	"time"
)

type Flags struct {
	// Global flags
	Local          bool // Use local vagrant test cluster
	JobPath        string
	ClusterPath    string
	TunnelOverride string
	Groups         []string
	ScalingGroup   uint
	DryRun         bool
	Force          bool
	StopDelay      time.Duration
	DestroyDelay   time.Duration
	SliceDelay     time.Duration
	Options        Options
}
