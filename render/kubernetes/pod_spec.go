package kubernetes

import (
	"encoding/json"

	k8s "github.com/YakLabs/k8s-client"
	"github.com/pulcy/j2/jobs"
)

const (
	RestartPolicyAlways    = "Always"
	RestartPolicyOnFailure = "OnFailure"
	RestartPolicyNever     = "Never"
)

const (
	PodInitContainersAnnotationKey = "pod.alpha.kubernetes.io/init-containers"
)

// createPodSpec creates a pod-spec for all tasks in a given pod.
func createPodSpec(tg *jobs.TaskGroup, pod pod, ctx generatorContext) (*k8s.PodSpec, map[string]string, error) {
	spec := &k8s.PodSpec{
		RestartPolicy: RestartPolicyAlways,
	}

	// Volumes
	volumes, err := createVolumes(tg, pod, ctx)
	if err != nil {
		return nil, nil, maskAny(err)
	}
	spec.Volumes = volumes

	// Containers
	annotations := make(map[string]string)
	for _, t := range pod.tasks {
		if t.Network.IsHost() {
			spec.HostNetwork = true
		}
		initContainers, containers, extraVols, err := createTaskContainers(t, pod, ctx)
		if err != nil {
			return nil, nil, maskAny(err)
		}
		if len(initContainers) > 0 {
			raw, err := json.Marshal(initContainers)
			if err != nil {
				return nil, nil, maskAny(err)
			}
			annotations[PodInitContainersAnnotationKey] = string(raw)
		}
		spec.Volumes = append(spec.Volumes, extraVols...)
		spec.Containers = append(spec.Containers, containers...)
	}

	return spec, annotations, nil
}
