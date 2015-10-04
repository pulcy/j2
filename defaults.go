package main

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
)

var (
	defaultGroups = []string{}
)
