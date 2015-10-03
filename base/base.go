package base

import (
	"arvika.pulcy.com/pulcy/deployit/flags"
	"arvika.pulcy.com/pulcy/deployit/jobs"
)

func NewBase(flags *flags.Flags) *jobs.ServiceGroup {
	sg := jobs.NewServiceGroup("base")
	sg.Add(newLoadBalancer(flags))
	sg.Add(newRegistrator(flags))
	return sg
}
