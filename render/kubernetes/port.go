package kubernetes

import (
	"github.com/pulcy/j2/jobs"

	"k8s.io/client-go/pkg/api/v1"
)

func createContainerPort(port jobs.PortMapping) (v1.ContainerPort, error) {
	p, err := port.Parse()
	if err != nil {
		return v1.ContainerPort{}, maskAny(err)
	}
	cp := v1.ContainerPort{
		ContainerPort: int32(p.ContainerPort),
		HostPort:      int32(p.HostPort),
		HostIP:        p.HostIP,
		Protocol:      v1.ProtocolTCP,
	}
	if p.IsUDP() {
		cp.Protocol = v1.ProtocolUDP
	}
	return cp, nil
}
