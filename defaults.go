package main

import (
	"time"
)

// globals
const (
	defaultJobPath        = ""
	defaultClusterPath    = ""
	defaultTunnelOverride = ""
	defaultForce          = false
	defaultDryRun         = false
	defaultScalingGroup   = uint(0)    // all
	defaultDomain         = "iggi.xyz" // TODO
	defaultLocal          = false
)

var (
	defaultGroups       = []string{}
	defaultStopDelay    = 15 * time.Second // sec
	defaultDestroyDelay = 15 * time.Second // sec
	defaultSliceDelay   = 30 * time.Second // sec
)
