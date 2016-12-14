package kubernetes

import (
	"encoding/json"

	"github.com/pulcy/j2/jobs"
	"k8s.io/client-go/pkg/api/v1"
)

// createPodTemplateSpec creates a podTemplateSpec for all tasks in a given pod.
func createPodTemplateSpec(tg *jobs.TaskGroup, pod pod, ctx generatorContext) (v1.PodTemplateSpec, error) {
	tspec := v1.PodTemplateSpec{}
	tspec.ObjectMeta.Name = pod.name
	setPodLabels(&tspec.ObjectMeta, tg, pod)

	spec, err := createPodSpec(tg, pod, ctx)
	if err != nil {
		return v1.PodTemplateSpec{}, maskAny(err)
	}
	tspec.Spec = spec

	// Affinity
	constraints := jobs.Constraints{}
	for _, t := range pod.tasks {
		constraints = constraints.Merge(t.MergedConstraints())
	}
	if constraints.Len() > 0 {
		a, err := createAffinity(constraints, tg, pod, ctx)
		if err != nil {
			return v1.PodTemplateSpec{}, maskAny(err)
		}
		raw, err := json.Marshal(a)
		if err != nil {
			return v1.PodTemplateSpec{}, maskAny(err)
		}
		setAnnotation(&tspec.ObjectMeta, v1.AffinityAnnotationKey, string(raw))
	}

	return tspec, nil
}
