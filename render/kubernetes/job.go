package kubernetes

import (
	k8s "github.com/YakLabs/k8s-client"
	"github.com/pulcy/j2/jobs"
)

// createJobs creates all jobs needed for the given task group.
func createJobs(tg *jobs.TaskGroup, pod pod, ctx generatorContext) ([]k8s.Job, error) {
	if pod.hasServiceTasks() || !pod.hasOneShotTasks() {
		// Job only takes oneshot tasks.
		return nil, nil
	}

	d := k8s.NewJob(ctx.Namespace, resourceName(pod.name, kindJob))
	d.Spec.Completions = 1
	d.Spec.Parallelism = 1
	setTaskGroupLabelsAnnotations(&d.ObjectMeta, tg)

	requireRestartPolicyAlways := false
	template, err := createPodTemplateSpec(tg, pod, ctx, requireRestartPolicyAlways)
	if err != nil {
		return nil, maskAny(err)
	}
	d.Spec.Template = *template

	return []k8s.Job{*d}, nil
}
