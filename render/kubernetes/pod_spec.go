package kubernetes

import (
	"github.com/pulcy/j2/jobs"
	"k8s.io/client-go/pkg/api/v1"
)

// createPodSpec creates a pod-spec for all tasks in a given pod.
func createPodSpec(tg *jobs.TaskGroup, pod pod, ctx generatorContext) (v1.PodSpec, error) {
	spec := v1.PodSpec{}
	spec.RestartPolicy = v1.RestartPolicyAlways

	// Volumes
	volumes, err := createVolumes(tg, pod, ctx)
	if err != nil {
		return v1.PodSpec{}, maskAny(err)
	}
	spec.Volumes = volumes

	// Containers
	for _, t := range pod.tasks {
		initContainers, containers, err := createTaskContainers(t, pod, ctx)
		if err != nil {
			return v1.PodSpec{}, maskAny(err)
		}
		spec.InitContainers = append(spec.InitContainers, initContainers...)
		spec.Containers = append(spec.Containers, containers...)
	}

	return spec, nil
}
