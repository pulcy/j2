package base

import (
	"arvika.pulcy.com/pulcy/deployit/flags"
	"arvika.pulcy.com/pulcy/deployit/services"
)

func newLoadBalancer(flags *flags.Flags) services.Service {
	s := services.NewDockerService("lb", "Load balancer").
		Global().
		Image(flags.PrivateRegistry, "load-balancer", flags.LoadBalancerVersion).
		Ports(
		services.NewPort("80").HostPort("80"),
		services.NewPort("443").HostPort("443"),
		services.NewPort("7086").HostPort("7086"),
	)
	return s
}
