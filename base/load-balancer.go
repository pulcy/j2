package base

import (
	"arvika.pulcy.com/pulcy/deployit/flags"
	"arvika.pulcy.com/pulcy/deployit/jobs"
)

func newLoadBalancer(flags *flags.Flags) jobs.Service {
	s := jobs.NewDockerService("lb", "Load balancer").
		Global().
		Image(flags.PrivateRegistry, "load-balancer", flags.LoadBalancerVersion).
		Ports(
		jobs.NewPort("80").HostPort("80"),
		jobs.NewPort("443").HostPort("443"),
		jobs.NewPort("7086").HostPort("7086"),
	)
	return s
}
