package kubernetes

import (
	"github.com/ericchiang/k8s"
	"github.com/ericchiang/k8s/api/v1"
	"github.com/pulcy/j2/jobs"
)

func createContainerPort(port jobs.PortMapping) (*v1.ContainerPort, error) {
	p, err := port.Parse()
	if err != nil {
		return nil, maskAny(err)
	}
	cp := &v1.ContainerPort{
		ContainerPort: k8s.Int32P(int32(p.ContainerPort)),
		Protocol:      k8s.StringP("TCP"),
	}
	if p.HasHostPort() {
		cp.HostPort = k8s.Int32P(int32(p.HostPort))
	}
	if p.HasHostIP() {
		cp.HostIP = k8s.StringP(p.HostIP)
	}
	if p.IsUDP() {
		cp.Protocol = k8s.StringP("UDP")
	}
	return cp, nil
}
