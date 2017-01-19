package kubernetes

import (
	"encoding/json"
	"fmt"

	k8s "github.com/YakLabs/k8s-client"
	"github.com/YakLabs/k8s-client/intstr"
	"github.com/pulcy/j2/cluster"
	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/pkg/robin"
	"github.com/pulcy/robin-api"
)

const (
	RobinFrontendRecordsAnnotationKey = "pulcy.com.robin.frontend.records"
)

// createIngresses creates all ingresses needed for the given task group.
func createIngresses(tg *jobs.TaskGroup, pod pod, ctx generatorContext) ([]k8s.Ingress, error) {
	var ingresses []k8s.Ingress
	for _, t := range pod.tasks {
		i, err := createFrontendIngress(t, tg, pod, ctx)
		if err != nil {
			return nil, maskAny(err)
		}
		if i != nil {
			ingresses = append(ingresses, *i)
		}
	}

	return ingresses, nil
}

// createFrontendIngress creates all an Ingress to serve the public & private frontends of the given task.
func createFrontendIngress(t *jobs.Task, tg *jobs.TaskGroup, pod pod, ctx generatorContext) (*k8s.Ingress, error) {
	if len(t.PublicFrontEnds) == 0 && len(t.PrivateFrontEnds) == 0 {
		return nil, nil
	}
	d := k8s.NewIngress(ctx.Namespace, taskIngressName(t))
	setTaskGroupLabelsAnnotations(&d.ObjectMeta, tg)

	// Public frontends
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

	// Private frontends
	for _, frontend := range t.PrivateFrontEnds {
		pathPrefix := "/"
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
			Host: taskServiceDNSName(t, ctx.Cluster.KubernetesOptions.Domain),
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

	publicOnly := false
	records, err := robin.CreateFrontEndRecords(t, 1, publicOnly, &ingressFrontendNameBuilder{ctx.Cluster, tg})
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

	return d, nil
}

type ingressFrontendNameBuilder struct {
	cluster cluster.Cluster
	tg      *jobs.TaskGroup
}

// Create the serviceName of the given task.
// This name is used in the Key of the returned records.
func (nb *ingressFrontendNameBuilder) CreateServiceName(t *jobs.Task) (string, error) {
	return taskServiceName(t), nil
}

// Create the name used in the Service field of the returned records.
func (nb *ingressFrontendNameBuilder) CreateTargetServiceName(t *jobs.Task) (string, error) {
	name, err := createTargetServiceName(t, nb.tg.Job())
	if err != nil {
		return "", maskAny(err)
	}
	return name, nil
}

// Create the Domain field of selectors created for private-frontends.
func (nb *ingressFrontendNameBuilder) CreatePrivateDomainNames(t *jobs.Task) ([]string, error) {
	return []string{taskServiceDNSName(t, nb.cluster.KubernetesOptions.Domain), t.PrivateDomainName()}, nil
}

// Create the Domain field of selectors created for instance specific private-frontends.
func (nb *ingressFrontendNameBuilder) CreateInstanceSpecificPrivateDomainNames(t *jobs.Task, instance uint) ([]string, error) {
	return nil, maskAny(fmt.Errorf("Instance specific private domain names are not supported"))
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
		if port == 0 {
			port = 80
		}
		port = target.PublicFrontEndPort(port)
	} else {
		serviceName = taskServiceName(t)
	}
	return &k8s.IngressBackend{
		ServiceName: serviceName,
		ServicePort: intstr.FromInt(port),
	}, nil
}
