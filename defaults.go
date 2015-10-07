package main

import (
	"time"
)

// globals
const (
	defaultJobPath         = ""
	defaultStack           = ""
	defaultTunnel          = ""
	defaultForce           = false
	defaultDryRun          = false
	defaultScalingGroup    = uint(0) // all
	defaultPrivateRegistry = "arvika.pulcy.com:5000"
	defaultLogLevel        = "debug"
	defaultDomain          = "iggi.xyz" // TODO
	defaultLocal           = false
)

var (
	defaultGroups       = []string{}
	defaultStopDelay    = 15 * time.Second // sec
	defaultDestroyDelay = 15 * time.Second // sec
	defaultSliceDelay   = 30 * time.Second // sec
)
