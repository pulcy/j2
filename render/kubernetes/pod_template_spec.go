package kubernetes

import (
	"encoding/json"

	k8s "github.com/YakLabs/k8s-client"
	"github.com/pulcy/j2/jobs"
)

// createPodTemplateSpec creates a podTemplateSpec for all tasks in a given pod.
func createPodTemplateSpec(tg *jobs.TaskGroup, pod pod, ctx generatorContext) (*k8s.PodTemplateSpec, error) {
	tspec := k8s.NewPodTemplateSpec(ctx.Namespace, pod.name)
	setPodLabels(&tspec.ObjectMeta, tg, pod)

	spec, annotations, err := createPodSpec(tg, pod, ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	tspec.Spec = spec
	for k, v := range annotations {
		setAnnotation(&tspec.ObjectMeta, k, v)
	}

	// Affinity
	constraints := jobs.Constraints{}
	for _, t := range pod.tasks {
		constraints = constraints.Merge(t.MergedConstraints())
	}
	if constraints.Len() > 0 {
		a, err := createAffinity(constraints, tg, pod, ctx)
		if err != nil {
			return nil, maskAny(err)
		}
		raw, err := json.Marshal(a)
		if err != nil {
			return nil, maskAny(err)
		}
		setAnnotation(&tspec.ObjectMeta, k8s.AffinityAnnotationKey, string(raw))
	}

	return tspec, nil
}
