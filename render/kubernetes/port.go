package kubernetes

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/client-go/pkg/api/v1"
)

func createContainerPort(port string) (v1.ContainerPort, error) {
	parts := strings.Split(port, ":")
	hostIP := ""
	containerPort := 0
	hostPort := 0
	var err error
	switch len(parts) {
	case 1:
		containerPort, err = strconv.Atoi(parts[0])
		if err != nil {
			return v1.ContainerPort{}, maskAny(err)
		}
	case 2:
		if parts[0] != "" {
			hostPort, err = strconv.Atoi(parts[0])
			if err != nil {
				return v1.ContainerPort{}, maskAny(err)
			}
		}
		containerPort, err = strconv.Atoi(parts[1])
		if err != nil {
			return v1.ContainerPort{}, maskAny(err)
		}
	case 3:
		if parts[0] != "" {
			hostIP = parts[0]
		}
		if parts[1] != "" {
			hostPort, err = strconv.Atoi(parts[1])
			if err != nil {
				return v1.ContainerPort{}, maskAny(err)
			}
		}
		containerPort, err = strconv.Atoi(parts[2])
		if err != nil {
			return v1.ContainerPort{}, maskAny(err)
		}
	default:
		return v1.ContainerPort{}, maskAny(fmt.Errorf("Unknown port format '%s'", port))
	}
	cp := v1.ContainerPort{
		ContainerPort: int32(containerPort),
		HostPort:      int32(hostPort),
		HostIP:        hostIP,
	}
	return cp, nil
}
