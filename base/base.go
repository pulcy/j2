package base

import (
	"arvika.pulcy.com/pulcy/deployit/flags"
	"arvika.pulcy.com/pulcy/deployit/services"
)

func NewBase(flags *flags.Flags) *services.ServiceGroup {
	sg := services.NewServiceGroup("base")
	sg.Add(newLoadBalancer(flags))
	sg.Add(newRegistrator(flags))
	return sg
}
