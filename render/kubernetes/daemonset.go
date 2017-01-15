package kubernetes

import (
	k8s "github.com/YakLabs/k8s-client"
	"github.com/pulcy/j2/jobs"
)

// createDaemonSets creates all daemon sets needed for the given task group.
func createDaemonSets(tg *jobs.TaskGroup, pod pod, ctx generatorContext) ([]k8s.DaemonSet, error) {
	if !tg.Global {
		// Non-global is mapped onto Deployments.
		return nil, nil
	}
	if !pod.hasServiceTasks() {
		// DaemonSet need at least 1 service task
		return nil, nil
	}

	d := k8s.NewDaemonSet(ctx.Namespace, resourceName(pod.name, kindDaemonSet))
	setTaskGroupLabelsAnnotations(&d.ObjectMeta, tg)

	requireRestartPolicyAlways := true
	template, err := createPodTemplateSpec(tg, pod, ctx, requireRestartPolicyAlways)
	if err != nil {
		return nil, maskAny(err)
	}
	d.Spec.Template = *template

	return []k8s.DaemonSet{*d}, nil
}
