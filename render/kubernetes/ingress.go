package kubernetes

import (
	"github.com/ericchiang/k8s"
	"github.com/ericchiang/k8s/api/v1"
	"github.com/ericchiang/k8s/apis/extensions/v1beta1"
	"github.com/pulcy/j2/jobs"
	pkg "github.com/pulcy/j2/pkg/kubernetes"
)

// createIngresses creates allingressesservices needed for the given task group.
func createIngresses(tg *jobs.TaskGroup, pod pod, ctx generatorContext) ([]v1beta1.Ingress, error) {
	d := v1beta1.Ingress{
		Metadata: &v1.ObjectMeta{
			Name:      k8s.StringP(resourceName(pod.name, kindIngress)),
			Namespace: k8s.StringP(ctx.Namespace),
		},
		Spec: &v1beta1.IngressSpec{},
	}
	setTaskGroupLabelsAnnotations(d.GetMetadata(), tg)

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
			rule := &v1beta1.IngressRule{
				Host: k8s.StringP(frontend.Domain),
				IngressRuleValue: &v1beta1.IngressRuleValue{
					Http: &v1beta1.HTTPIngressRuleValue{
						Paths: []*v1beta1.HTTPIngressPath{
							&v1beta1.HTTPIngressPath{
								Path:    k8s.StringP(pathPrefix),
								Backend: backend,
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
	servicePort := pkg.FromInt(int32(port))
	return &v1beta1.IngressBackend{
		ServiceName: k8s.StringP(resourceName(pod.name, kindService)),
		ServicePort: &servicePort,
	}
}
