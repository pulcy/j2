package base

import (
	"arvika.pulcy.com/pulcy/deployit/flags"
	"arvika.pulcy.com/pulcy/deployit/jobs"
)

func newStunnelPem(flags *flags.Flags) jobs.Service {
	s := jobs.NewDockerService("stunnel-pem", "Stunnel certificates").
		Global().
		Image(flags.PrivateRegistry, "stunnel-pem", flags.StunnelPemVersion).
		Environment("PASSPHRASE", flags.StunnelPemPassphrase)
	return s
}

func newStunnelRegistryClient(flags *flags.Flags) jobs.Service {
	s := jobs.NewDockerService("stunnel-registry-client", "Stunnel registry client").
		Global().
		Image(flags.PrivateRegistry, "stunnel-registry-client", flags.StunnelRegistryClientVersion).
		VolumesFrom("stunnel-pem").
		Ports(jobs.NewPort("5000").HostPort("5000"))
	return s
}
