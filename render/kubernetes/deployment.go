package kubernetes

import (
	"github.com/pulcy/j2/jobs"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
)

// createDeployments creates all deployments needed for the given task group.
func createDeployments(tg *jobs.TaskGroup, pod pod, ctx generatorContext) ([]v1beta1.Deployment, error) {
	if tg.Global {
		// Global is mapped onto DaemonSets.
		return nil, nil
	}

	d := v1beta1.Deployment{}
	d.TypeMeta.Kind = "Deployment"
	d.TypeMeta.APIVersion = "extensions/v1beta1"

	d.ObjectMeta.Name = resourceName(pod.name, kindDeployment)
	d.ObjectMeta.Namespace = ctx.Namespace
	setTaskGroupLabelsAnnotations(&d.ObjectMeta, tg)

	count := int32(tg.Count)
	d.Spec.Replicas = &count

	template, err := createPodTemplateSpec(tg, pod, ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	d.Spec.Template = template

	return []v1beta1.Deployment{d}, nil
}

type deploymentResource struct {
	unitData
	resource v1beta1.Deployment
}

// Namespace returns the namespace the resource should be added to.
func (r *deploymentResource) Namespace() string {
	return r.resource.ObjectMeta.Namespace
}

// Start creates/updates the deployment
func (r *deploymentResource) Start(cs *kubernetes.Clientset) error {
	api := cs.Deployments(r.Namespace())
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
