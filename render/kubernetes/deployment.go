package kubernetes

import (
	"github.com/pulcy/j2/jobs"

	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
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
