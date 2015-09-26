package flags

type Flags struct {
	// Global flags
	Service         string
	Stack           string
	Tunnel          string
	ScalingGroup    uint8
	DryRun          bool
	Force           bool
	PrivateRegistry string
	LogLevel        string
	DefaultScale    uint8

	// Image versions
	LoadBalancerVersion string
}
