package kubernetes

import (
	"github.com/pulcy/j2/jobs"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
)

// createServices creates all services needed for the given task group.
func createServices(tg *jobs.TaskGroup, pod pod, ctx generatorContext) ([]v1.Service, error) {
	var services []v1.Service
	for _, t := range pod.tasks {
		d := v1.Service{}
		d.TypeMeta.Kind = "Service"
		d.TypeMeta.APIVersion = "v1"

		d.ObjectMeta.Name = taskServiceName(t)
		d.ObjectMeta.Namespace = ctx.Namespace
		setTaskGroupLabelsAnnotations(&d.ObjectMeta, tg)

		d.Spec.Selector = createPodSelector(d.Spec.Selector, pod)
		for _, p := range t.Ports {
			pp, err := p.Parse()
			if err != nil {
				return nil, maskAny(err)
			}
			servicePort := v1.ServicePort{
				Port:     int32(pp.ContainerPort),
				Protocol: v1.ProtocolTCP,
			}
			if pp.IsUDP() {
				servicePort.Protocol = v1.ProtocolUDP
			}
			d.Spec.Ports = append(d.Spec.Ports, servicePort)
		}
		services = append(services, d)
	}

	return services, nil
}

type serviceResource struct {
	unitData
	resource v1.Service
}

// Namespace returns the namespace the resource should be added to.
func (r *serviceResource) Namespace() string {
	return r.resource.ObjectMeta.Namespace
}

// Start creates/updates the service
func (r *serviceResource) Start(cs *kubernetes.Clientset) error {
	api := cs.Services(r.Namespace())
	_, err := api.Get(r.resource.ObjectMeta.Name, metav1.GetOptions{})
	if err == nil {
		// Update
		if _, err := api.Update(&r.resource); err != nil {
			return maskAny(err)
		}
	} else {
		// Create
		if _, err := api.Create(&r.resource); err != nil {
			return maskAny(err)
		}
	}
	return nil
}
