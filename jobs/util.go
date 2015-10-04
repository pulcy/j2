package jobs

import (
	"arvika.pulcy.com/pulcy/deployit/units"
)

// groupUnitsOnMachine modifies given list of units such
// that all units are forced on the same machine (of the first unit)
func groupUnitsOnMachine(units []*units.Unit) {
	for i, u := range units {
		if i == 0 {
			continue
		}
		u.ExecOptions.MachineOf = units[0].FullName
	}
}
