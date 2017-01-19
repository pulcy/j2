package kubernetes

import (
	"fmt"
	"strconv"
	"strings"

	k8s "github.com/YakLabs/k8s-client"
	"github.com/pulcy/j2/jobs"
	pkg "github.com/pulcy/j2/pkg/kubernetes"
)

// createServices creates all services needed for the given task group.
func createServices(tg *jobs.TaskGroup, pod pod, ctx generatorContext) ([]k8s.Service, error) {
	var services []k8s.Service
	for _, t := range pod.tasks {
		if t.Type.IsProxy() {
			// If the proxy has rewriting, then the actual rewriting is done by the robin load-balancer.
			// We create a service that selects the load-balancer pods.
			s, err := createProxyService(tg, t, pod, ctx)
			if err != nil {
				return nil, maskAny(err)
			}
			services = append(services, s)
		} else if t.Type.IsService() {
			if len(t.Ports) > 0 || len(t.PublicFrontEnds) > 0 || len(t.PrivateFrontEnds) > 0 {
				// Normal service for the task
				taskServices, err := createPodServices(tg, t, pod, ctx)
				if err != nil {
					return nil, maskAny(err)
				}
				services = append(services, taskServices...)
			}
		}
	}

	return services, nil
}

// createPodServices creates all services needed for a specific task that runs a standard pod.
func createPodServices(tg *jobs.TaskGroup, t *jobs.Task, pod pod, ctx generatorContext) ([]k8s.Service, error) {
	var services []k8s.Service

	// Normal service for the task
	d := newService(ctx.Namespace, taskServiceName(t))
	setTaskGroupLabelsAnnotations(&d.ObjectMeta, tg)
	d.Spec.Selector = createPodSelector(d.Spec.Selector, pod)

	ports := collectPorts(t)
	for _, p := range ports {
		pp, err := p.Parse()
		if err != nil {
			return nil, maskAny(err)
		}
		protocol := pp.ProtocolString()
		servicePort := k8s.ServicePort{
			Name:       strings.ToLower(fmt.Sprintf("%d-%s", pp.ContainerPort, protocol)),
			Port:       int32(pp.ContainerPort),
			Protocol:   k8s.Protocol(protocol),
			TargetPort: pkg.FromInt(int32(pp.ContainerPort)),
		}
		d.Spec.Ports = append(d.Spec.Ports, servicePort)
	}
	services = append(services, *d)

	// Host mapped ports (a service with all ports mapped to all hosts using a NodePort)
	if !t.Network.IsHost() {
		hmPorts := getHostMappedPorts(t.Ports)
		if len(hmPorts) > 0 {
			d := newService(ctx.Namespace, taskServiceName(t)+"-host")
			setTaskGroupLabelsAnnotations(&d.ObjectMeta, tg)
			d.Spec.Type = k8s.ServiceTypeNodePort
			d.Spec.Selector = createPodSelector(d.Spec.Selector, pod)

			for _, p := range hmPorts {
				pp, err := p.Parse()
				if err != nil {
					return nil, maskAny(err)
				}
				protocol := pp.ProtocolString()
				hostPort := pp.ContainerPort
				if pp.HasHostPort() {
					hostPort = pp.HostPort
				}
				servicePort := k8s.ServicePort{
					Name:       strings.ToLower(fmt.Sprintf("%d-%s", pp.ContainerPort, protocol)),
					Port:       int32(pp.ContainerPort),
					Protocol:   k8s.Protocol(protocol),
					TargetPort: pkg.FromInt(int32(pp.ContainerPort)),
					NodePort:   int32(hostPort),
				}
				d.Spec.Ports = append(d.Spec.Ports, servicePort)
			}
			services = append(services, *d)
		}
	}

	return services, nil
}

// createProxyService create a service with the name of the given task.
// The selector of the service is not the pods of the task (because there are none).
// If the proxy has rewrite rules, the selector is the load-balancer (that does the rewriting)
// otherwise the selector is the target service itself.
func createProxyService(tg *jobs.TaskGroup, t *jobs.Task, pod pod, ctx generatorContext) (k8s.Service, error) {
	d := newService(ctx.Namespace, taskServiceName(t))
	setTaskGroupLabelsAnnotations(&d.ObjectMeta, tg)
	d.Spec.Type = k8s.ServiceTypeExternalName
	if t.Rewrite != nil {
		// Proxy traffic through load-balancer that will do the rewriting
		d.Spec.ExternalName = fmt.Sprintf("%s.svc.%s", pkg.LoadBalancerDNS, ctx.Cluster.KubernetesOptions.Domain)
	} else {
		// Proxy traffic directly to target service
		target, err := t.Target.Resolve(tg.Job())
		if err != nil {
			return k8s.Service{}, maskAny(err)
		}
		d.Spec.ExternalName = taskServiceDNSName(target, ctx.Cluster.KubernetesOptions.Domain)
	}

	return *d, nil
}

// collectPorts creates a list of port-mappings for all ports
// created by the given task.
// That is `ports`, `private-frontend.port` && `frontend.port`.
func collectPorts(t *jobs.Task) []jobs.PortMapping {
	ports := t.Ports
	portFound := func(containerPort int) bool {
		for _, p := range ports {
			pm, _ := p.Parse()
			if pm.ContainerPort == containerPort {
				return true
			}
		}
		return false
	}
	for _, f := range t.PublicFrontEnds {
		if f.Port != 0 {
			if !portFound(f.Port) {
				port := strconv.Itoa(f.Port)
				ports = append(ports, jobs.PortMapping(port))
			}
		}
	}
	for _, f := range t.PrivateFrontEnds {
		if f.Port != 0 {
			if !portFound(f.Port) {
				port := strconv.Itoa(f.Port)
				ports = append(ports, jobs.PortMapping(port))
			}
		}
	}
	return ports
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

func newService(namespace, name string) *k8s.Service {
	s := k8s.NewService(namespace, name)
	s.Spec.Type = k8s.ServiceTypeClusterIP
	s.Spec.SessionAffinity = "None"
	return s
}
