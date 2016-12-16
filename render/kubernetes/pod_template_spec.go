package kubernetes

import (
	"encoding/json"

	"github.com/ericchiang/k8s"
	"github.com/ericchiang/k8s/api/v1"
	"github.com/pulcy/j2/jobs"
)

const (
	AffinityAnnotationKey = "scheduler.alpha.kubernetes.io/affinity"
)

// createPodTemplateSpec creates a podTemplateSpec for all tasks in a given pod.
func createPodTemplateSpec(tg *jobs.TaskGroup, pod pod, ctx generatorContext) (*v1.PodTemplateSpec, error) {
	tspec := &v1.PodTemplateSpec{
		Metadata: &v1.ObjectMeta{
			Name: k8s.StringP(pod.name),
		},
	}
	setPodLabels(tspec.GetMetadata(), tg, pod)

	spec, annotations, err := createPodSpec(tg, pod, ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	tspec.Spec = spec
	for k, v := range annotations {
		setAnnotation(tspec.GetMetadata(), k, v)
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
		setAnnotation(tspec.GetMetadata(), AffinityAnnotationKey, string(raw))
	}

	return tspec, nil
}
