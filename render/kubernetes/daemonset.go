package kubernetes

import (
	"github.com/pulcy/j2/jobs"

	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

// createDaemonSets creates all daemon sets needed for the given task group.
func createDaemonSets(tg *jobs.TaskGroup, pod pod, ctx generatorContext) ([]v1beta1.DaemonSet, error) {
	if !tg.Global {
		// Non-global is mapped onto Deployments.
		return nil, nil
	}

	d := v1beta1.DaemonSet{}
	d.TypeMeta.Kind = "DaemonSet"
	d.TypeMeta.APIVersion = "extensions/v1beta1"

	d.ObjectMeta.Name = resourceName(pod.name, kindDaemonSet)
	d.ObjectMeta.Namespace = ctx.Namespace
	setTaskGroupLabelsAnnotations(&d.ObjectMeta, tg)

	template, err := createPodTemplateSpec(tg, pod, ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	d.Spec.Template = template

	return []v1beta1.DaemonSet{d}, nil
}
