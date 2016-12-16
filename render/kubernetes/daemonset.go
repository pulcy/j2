package kubernetes

import (
	"github.com/ericchiang/k8s"
	"github.com/ericchiang/k8s/api/v1"
	"github.com/ericchiang/k8s/apis/extensions/v1beta1"
	"github.com/pulcy/j2/jobs"
)

// createDaemonSets creates all daemon sets needed for the given task group.
func createDaemonSets(tg *jobs.TaskGroup, pod pod, ctx generatorContext) ([]v1beta1.DaemonSet, error) {
	if !tg.Global {
		// Non-global is mapped onto Deployments.
		return nil, nil
	}

	d := v1beta1.DaemonSet{
		Metadata: &v1.ObjectMeta{
			Name:      k8s.StringP(resourceName(pod.name, kindDaemonSet)),
			Namespace: k8s.StringP(ctx.Namespace),
		},
	}
	setTaskGroupLabelsAnnotations(d.GetMetadata(), tg)

	template, err := createPodTemplateSpec(tg, pod, ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	d.Spec.Template = template

	return []v1beta1.DaemonSet{d}, nil
}
