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

// GroupUnitsOnMachine modifies given list of units such
// that all units are forced on the same machine (of the first unit)
func GroupUnitsOnMachine(units ...*Unit) {
	for i, u := range units {
		if i == 0 {
			continue
		}
		u.ExecOptions.MachineOf = units[0].FullName
	}
}
