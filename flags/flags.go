package flags

type Flags struct {
	// Global flags
	JobPath         string
	Groups          []string
	Stack           string
	Tunnel          string
	ScalingGroup    uint8
	DryRun          bool
	Force           bool
	PrivateRegistry string
	LogLevel        string
}
