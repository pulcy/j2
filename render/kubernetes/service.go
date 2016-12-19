package kubernetes

import (
	k8s "github.com/YakLabs/k8s-client"
	"github.com/pulcy/j2/jobs"
)

// createServices creates all services needed for the given task group.
func createServices(tg *jobs.TaskGroup, pod pod, ctx generatorContext) ([]k8s.Service, error) {
	var services []k8s.Service
	for _, t := range pod.tasks {
		d := k8s.NewService(ctx.Namespace, taskServiceName(t))
		setTaskGroupLabelsAnnotations(&d.ObjectMeta, tg)

		d.Spec.Selector = createPodSelector(d.Spec.Selector, pod)
		for _, p := range t.Ports {
			pp, err := p.Parse()
			if err != nil {
				return nil, maskAny(err)
			}
			servicePort := k8s.ServicePort{
				Port:     int32(pp.ContainerPort),
				Protocol: "TCP",
			}
			if pp.IsUDP() {
				servicePort.Protocol = "UDP"
			}
			d.Spec.Ports = append(d.Spec.Ports, servicePort)
		}
		services = append(services, *d)
	}

	return services, nil
}
