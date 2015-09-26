package services

type ServiceGroup struct {
	Name     string
	Services []Service
}

func NewServiceGroup(name string) *ServiceGroup {
	return &ServiceGroup{
		Name: name,
	}
}

func (sg *ServiceGroup) Add(s ...Service) *ServiceGroup {
	sg.Services = append(sg.Services, s...)
	return sg
}

func (sg *ServiceGroup) Generate(service string, currentScalingGroup uint8) *Generator {
	return newGenerator(sg, service, currentScalingGroup)
}
