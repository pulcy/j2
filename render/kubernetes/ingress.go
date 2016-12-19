package kubernetes

import (
	k8s "github.com/YakLabs/k8s-client"
	"github.com/YakLabs/k8s-client/intstr"
	"github.com/pulcy/j2/jobs"
)

// createIngresses creates allingressesservices needed for the given task group.
func createIngresses(tg *jobs.TaskGroup, pod pod, ctx generatorContext) ([]k8s.Ingress, error) {
	d := k8s.NewIngress(ctx.Namespace, resourceName(pod.name, kindIngress))
	setTaskGroupLabelsAnnotations(&d.ObjectMeta, tg)

	port2Backend := map[int]*k8s.IngressBackend{}
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
			rule := k8s.IngressRule{
				Host: frontend.Domain,
				HTTP: &k8s.HTTPIngressRuleValue{
					Paths: []k8s.HTTPIngressPath{
						k8s.HTTPIngressPath{
							Path:    pathPrefix,
							Backend: *backend,
						},
					},
				},
			}
			// TODO users, rewrite, mode, ...
			d.Spec.Rules = append(d.Spec.Rules, rule)
		}
	}

	return []k8s.Ingress{*d}, nil
}

func createIngressBackend(pod pod, port int) *k8s.IngressBackend {
	return &k8s.IngressBackend{
		ServiceName: resourceName(pod.name, kindService),
		ServicePort: intstr.FromInt(port),
	}
}
