package kubernetes

import (
	k8s "github.com/YakLabs/k8s-client"
	"github.com/YakLabs/k8s-client/intstr"
	"github.com/pulcy/j2/jobs"
)

// createDeployments creates all deployments needed for the given task group.
func createDeployments(tg *jobs.TaskGroup, pod pod, ctx generatorContext) ([]k8s.Deployment, error) {
	if tg.Global {
		// Global is mapped onto DaemonSets.
		return nil, nil
	}
	if !pod.hasServiceTasks() {
		// Deployments need at least 1 service task
		return nil, nil
	}

	maxSurge := int(tg.Count)
	if pod.hasRWHostVolumes() {
		// Since we use host mapped volumes, make sure that no 2 pods use the the same volume
		// at the same time.
		maxSurge = 0
	}

	d := k8s.NewDeployment(ctx.Namespace, resourceName(pod.name, kindDeployment))
	d.Spec.Replicas = int(tg.Count)
	d.Spec.Strategy = &k8s.DeploymentStrategy{
		Type: k8s.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &k8s.RollingUpdateDeployment{
			MaxUnavailable: intstr.FromInt(1),
			MaxSurge:       intstr.FromInt(maxSurge),
		},
	}
	setTaskGroupLabelsAnnotations(&d.ObjectMeta, tg)

	requireRestartPolicyAlways := true
	template, err := createPodTemplateSpec(tg, pod, ctx, requireRestartPolicyAlways)
	if err != nil {
		return nil, maskAny(err)
	}
	d.Spec.Template = *template

	return []k8s.Deployment{*d}, nil
}
