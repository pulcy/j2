package units

type Unit struct {
	Name         string // e.g. "foo"
	FullName     string // e.g. "foo@1.service"
	Type         string "service|socket|timer"
	Description  string
	Scalable     bool
	ScalingGroup uint
	ExecOptions  *execOptions
	FleetOptions *fleetOptions
}
