package kubernetes

import (
	k8s "github.com/YakLabs/k8s-client"
	"github.com/pulcy/j2/jobs"
)

// createDeployments creates all deployments needed for the given task group.
func createDeployments(tg *jobs.TaskGroup, pod pod, ctx generatorContext) ([]k8s.Deployment, error) {
	if tg.Global {
		// Global is mapped onto DaemonSets.
		return nil, nil
	}

	d := k8s.NewDeployment(ctx.Namespace, resourceName(pod.name, kindDeployment))
	d.Spec.Replicas = int(tg.Count)
	setTaskGroupLabelsAnnotations(&d.ObjectMeta, tg)

	template, err := createPodTemplateSpec(tg, pod, ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	d.Spec.Template = *template

	return []k8s.Deployment{*d}, nil
}
