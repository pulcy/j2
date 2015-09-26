package base

import (
	"arvika.pulcy.com/iggi/deployit/flags"
	"arvika.pulcy.com/iggi/deployit/services"
)

func NewBase(flags *flags.Flags) *services.ServiceGroup {
	sg := services.NewServiceGroup("base")
	sg.Add(newLoadBalancer(flags))
	return sg
}

func newLoadBalancer(flags *flags.Flags) services.Service {
	s := services.NewDockerService("lb", "Load balancer").
		Global().
		Image(flags.PrivateRegistry, "load-balancer", flags.LoadBalancerVersion).
		Ports(services.NewPort("80").HostPort("80"))
	return s
}
