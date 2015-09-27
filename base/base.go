package base

import (
	"arvika.pulcy.com/iggi/deployit/flags"
	"arvika.pulcy.com/iggi/deployit/services"
)

func NewBase(flags *flags.Flags) *services.ServiceGroup {
	sg := services.NewServiceGroup("base")
	sg.Add(newLoadBalancer(flags))
	sg.Add(newRegistrator(flags))
	return sg
}
