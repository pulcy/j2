package units

import (
	"fmt"
)

type Unit struct {
	Name         string
	Type         string "service|socket|timer"
	Description  string
	Scalable     bool
	ScalingGroup uint8
	ExecOptions  *execOptions
	FleetOptions *fleetOptions
}

func (u *Unit) FullName() string {
	var fileName string
	if !u.Scalable {
		// shouldn't be scaled at all
		fileName = fmt.Sprintf("%s.%s", u.Name, u.Type)
	} else {
		fileName = fmt.Sprintf("%s@%d.%s", u.Name, u.ScalingGroup, u.Type)
	}
	return fileName
}

func (u *Unit) ContainerName() string {
	if !u.Scalable {
		return "%p.service"
	} else {
		return "%p-%i.service"
	}
}
