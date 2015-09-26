package units

type Unit struct {
	Name         string
	Type         string "service|socket|timer"
	Description  string
	DockerImage  string
	Scale        uint8
	ExecOptions  *execOptions
	FleetOptions *fleetOptions
}

func (u *Unit) FullName(num uint8) string {
	var fileName string
	if u.Scale == 0 {
		// shouldn't be scaled at all
		fileName = fmt.Sprintf("%s.%s", u.Name, u.Type)
	} else {
		fileName = fmt.Sprintf("%s@%d.%s", u.Name, num, u.Type)
	}
	return fileName
}

func (u *Unit) ContainerName() string {
	if u.Scale == 0 {
		return "%p.service"
	} else {
		return "%p-%i.service"
	}
}
