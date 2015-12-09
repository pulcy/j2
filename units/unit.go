package units

type Unit struct {
	Name            string // e.g. "foo"
	FullName        string // e.g. "foo@1.service"
	Type            string "service|socket|timer"
	Description     string
	Scalable        bool
	ScalingGroup    uint
	ExecOptions     *execOptions
	FleetOptions    *fleetOptions
	projectSettings map[string]string
}

// ProjectSetting as a key-value to the `X-<project>` section
func (unit *Unit) ProjectSetting(key, value string) {
	if unit.projectSettings == nil {
		unit.projectSettings = make(map[string]string)
	}
	unit.projectSettings[key] = value
}
