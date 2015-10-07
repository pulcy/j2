package flags

type Flags struct {
	// Global flags
	Local           bool // Use local vagrant test cluster
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
