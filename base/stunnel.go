package base

import (
	"arvika.pulcy.com/pulcy/deployit/flags"
	"arvika.pulcy.com/pulcy/deployit/services"
)

func newStunnelPem(flags *flags.Flags) services.Service {
	s := services.NewDockerService("stunnel-pem", "Stunnel certificates").
		Global().
		Image(flags.PrivateRegistry, "stunnel-pem", flags.StunnelPemVersion).
		Environment("PASSPHRASE", flags.StunnelPemPassphrase)
	return s
}

func newStunnelRegistryClient(flags *flags.Flags) services.Service {
	s := services.NewDockerService("stunnel-registry-client", "Stunnel registry client").
		Global().
		Image(flags.PrivateRegistry, "stunnel-registry-client", flags.StunnelRegistryClientVersion).
		VolumesFrom("stunnel-pem").
		Ports(services.NewPort("5000").HostPort("5000"))
	return s
}
