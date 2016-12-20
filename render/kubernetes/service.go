package kubernetes

import (
	"fmt"
	"strings"

	k8s "github.com/YakLabs/k8s-client"
	"github.com/pulcy/j2/jobs"
)

// createServices creates all services needed for the given task group.
func createServices(tg *jobs.TaskGroup, pod pod, ctx generatorContext) ([]k8s.Service, error) {
	var services []k8s.Service
	for _, t := range pod.tasks {
		// Normal service for the task
		if len(t.Ports) > 0 {
			d := k8s.NewService(ctx.Namespace, taskServiceName(t))
			setTaskGroupLabelsAnnotations(&d.ObjectMeta, tg)

			d.Spec.Selector = createPodSelector(d.Spec.Selector, pod)
			for _, p := range t.Ports {
				pp, err := p.Parse()
				if err != nil {
					return nil, maskAny(err)
				}
				protocol := pp.ProtocolString()
				servicePort := k8s.ServicePort{
					Name:     strings.ToLower(fmt.Sprintf("%d-%s", pp.ContainerPort, protocol)),
					Port:     int32(pp.ContainerPort),
					Protocol: k8s.Protocol(protocol),
				}
				d.Spec.Ports = append(d.Spec.Ports, servicePort)
			}
			services = append(services, *d)
		}
	}

	return services, nil
}

// getHostMappedPorts returns all port mappings from the given list that have their HostIP set to '0.0.0.0'.
func getHostMappedPorts(mappings []jobs.PortMapping) []jobs.PortMapping {
	var result []jobs.PortMapping
	for _, m := range mappings {
		if pm, err := m.Parse(); err == nil && pm.HostIP == "0.0.0.0" {
			result = append(result, m)
		}
	}
	return result
}
