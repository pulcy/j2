package base

import (
	"arvika.pulcy.com/pulcy/deployit/flags"
	"arvika.pulcy.com/pulcy/deployit/jobs"
)

func newRegistrator(flags *flags.Flags) jobs.Service {
	s := jobs.NewDockerService("registrator", "Service registrator").
		Global().
		Image("", "gliderlabs/registrator", flags.RegistratorVersion).
		Volume("/var/run/docker.sock", "/tmp/docker.sock").
		Args("-ip", "${COREOS_PRIVATE_IPV4}").
		Args("-ttl", "40").
		Args("-ttl-refresh", "20").
		Args("etcd://${COREOS_PRIVATE_IPV4}:4001/iggi")
	return s
}
