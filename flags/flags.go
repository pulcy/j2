package flags

type Flags struct {
	// Global flags
	JobPath         string
	Groups          []string
	Stack           string
	Domain          string
	Tunnel          string
	ScalingGroup    uint
	DryRun          bool
	Force           bool
	PrivateRegistry string
	LogLevel        string
}
