package kubernetes

import (
	"github.com/ericchiang/k8s"
	"github.com/ericchiang/k8s/api/v1"
	"github.com/pulcy/j2/jobs"
)

// createServices creates all services needed for the given task group.
func createServices(tg *jobs.TaskGroup, pod pod, ctx generatorContext) ([]v1.Service, error) {
	var services []v1.Service
	for _, t := range pod.tasks {
		d := v1.Service{
			Metadata: &v1.ObjectMeta{
				Name:      k8s.StringP(taskServiceName(t)),
				Namespace: k8s.StringP(ctx.Namespace),
			},
			Spec: &v1.ServiceSpec{},
		}
		setTaskGroupLabelsAnnotations(d.GetMetadata(), tg)

		d.Spec.Selector = createPodSelector(d.Spec.Selector, pod)
		for _, p := range t.Ports {
			pp, err := p.Parse()
			if err != nil {
				return nil, maskAny(err)
			}
			servicePort := &v1.ServicePort{
				Port:     k8s.Int32P(int32(pp.ContainerPort)),
				Protocol: k8s.StringP("TCP"),
			}
			if pp.IsUDP() {
				servicePort.Protocol = k8s.StringP("UDP")
			}
			d.Spec.Ports = append(d.Spec.Ports, servicePort)
		}
		services = append(services, d)
	}

	return services, nil
}
