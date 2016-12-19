package kubernetes

import (
	k8s "github.com/YakLabs/k8s-client"
	"github.com/pulcy/j2/jobs"
)

func createContainerPort(port jobs.PortMapping) (k8s.ContainerPort, error) {
	p, err := port.Parse()
	if err != nil {
		return k8s.ContainerPort{}, maskAny(err)
	}
	cp := k8s.ContainerPort{
		ContainerPort: p.ContainerPort,
		Protocol:      "TCP",
	}
	if p.HasHostPort() {
		cp.HostPort = p.HostPort
	}
	if p.HasHostIP() {
		cp.HostIP = p.HostIP
	}
	if p.IsUDP() {
		cp.Protocol = "UDP"
	}
	return cp, nil
}
