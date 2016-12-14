package kubernetes

import (
	"github.com/pulcy/j2/jobs"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/util/intstr"
)

// createIngresses creates allingressesservices needed for the given task group.
func createIngresses(tg *jobs.TaskGroup, pod pod, ctx generatorContext) ([]v1beta1.Ingress, error) {
	d := v1beta1.Ingress{}
	d.TypeMeta.Kind = "Ingress"
	d.TypeMeta.APIVersion = "extensions/v1beta1"

	d.ObjectMeta.Name = resourceName(pod.name, kindIngress)
	d.ObjectMeta.Namespace = ctx.Namespace
	setTaskGroupLabelsAnnotations(&d.ObjectMeta, tg)

	port2Backend := map[int]*v1beta1.IngressBackend{}
	for _, t := range pod.tasks {
		for _, frontend := range t.PublicFrontEnds {
			pathPrefix := frontend.PathPrefix
			if pathPrefix == "" {
				pathPrefix = "/"
			}
			port := frontend.Port
			backend, ok := port2Backend[port]
			if !ok {
				backend = createIngressBackend(pod, port)
				port2Backend[port] = backend
			}
			rule := v1beta1.IngressRule{
				Host: frontend.Domain,
				IngressRuleValue: v1beta1.IngressRuleValue{
					HTTP: &v1beta1.HTTPIngressRuleValue{
						Paths: []v1beta1.HTTPIngressPath{
							v1beta1.HTTPIngressPath{
								Path:    pathPrefix,
								Backend: *backend,
							},
						},
					},
				},
			}
			// TODO users, rewrite, mode, ...
			d.Spec.Rules = append(d.Spec.Rules, rule)
		}
	}

	return []v1beta1.Ingress{d}, nil
}

func createIngressBackend(pod pod, port int) *v1beta1.IngressBackend {
	return &v1beta1.IngressBackend{
		ServiceName: resourceName(pod.name, kindService),
		ServicePort: intstr.FromInt(port),
	}
}

type ingressResource struct {
	unitData
	resource v1beta1.Ingress
}

// Namespace returns the namespace the resource should be added to.
func (r *ingressResource) Namespace() string {
	return r.resource.ObjectMeta.Namespace
}

// Start creates/updates the service
func (r *ingressResource) Start(cs *kubernetes.Clientset) error {
	api := cs.Ingresses(r.Namespace())
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
