package kubernetes

import (
	"encoding/json"

	k8s "github.com/YakLabs/k8s-client"
	"github.com/YakLabs/k8s-client/intstr"
	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/pkg/robin"
	"github.com/pulcy/robin-api"
)

const (
	RobinFrontendRecordsAnnotationKey = "pulcy.com.robin.frontend.records"
)

// createIngresses creates allingressesservices needed for the given task group.
func createIngresses(tg *jobs.TaskGroup, pod pod, ctx generatorContext) ([]k8s.Ingress, error) {
	var ingresses []k8s.Ingress
	for _, t := range pod.tasks {
		if len(t.PublicFrontEnds) == 0 {
			continue
		}
		d := k8s.NewIngress(ctx.Namespace, taskIngressName(t))
		setTaskGroupLabelsAnnotations(&d.ObjectMeta, tg)

		port2Backend := map[int]*k8s.IngressBackend{}
		for _, frontend := range t.PublicFrontEnds {
			pathPrefix := frontend.PathPrefix
			if pathPrefix == "" {
				pathPrefix = "/"
			}
			port := frontend.Port
			backend, ok := port2Backend[port]
			if !ok {
				var err error
				backend, err = createIngressBackend(t, port, tg.Job())
				if err != nil {
					return nil, maskAny(err)
				}
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
			// Special features like users, rewrite & mode are supported through a
			// robin specific annotation. See below.
			d.Spec.Rules = append(d.Spec.Rules, rule)

		}
		publicOnly := true
		serviceName := taskServiceName(t)
		targetServiceName, err := createTargetServiceName(t, tg.Job())
		records, err := robin.CreateFrontEndRecords(t, 1, publicOnly, serviceName, targetServiceName)
		if err != nil {
			return nil, maskAny(err)
		}
		if len(records) > 0 {
			var apiRecords []api.FrontendRecord
			for _, r := range records {
				apiRecords = append(apiRecords, r.Record)
			}
			raw, err := json.Marshal(apiRecords)
			if err != nil {
				return nil, maskAny(err)
			}
			d.ObjectMeta.Annotations[RobinFrontendRecordsAnnotationKey] = string(raw)
		}

		ingresses = append(ingresses, *d)
	}

	return ingresses, nil
}

func createTargetServiceName(t *jobs.Task, job *jobs.Job) (string, error) {
	if t.Type.IsProxy() {
		target, err := t.Target.Resolve(job)
		if err != nil {
			return "", maskAny(err)
		}
		return taskServiceName(target), nil
	} else {
		return taskServiceName(t), nil
	}
}

func createIngressBackend(t *jobs.Task, port int, job *jobs.Job) (*k8s.IngressBackend, error) {
	var serviceName string
	if t.Type.IsProxy() {
		target, err := t.Target.Resolve(job)
		if err != nil {
			return nil, maskAny(err)
		}
		serviceName = taskServiceName(target)
		port = target.PublicFrontEndPort(80)
	} else {
		serviceName = taskServiceName(t)
	}
	return &k8s.IngressBackend{
		ServiceName: serviceName,
		ServicePort: intstr.FromInt(port),
	}, nil
}
