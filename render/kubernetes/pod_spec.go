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
	podInitContainersAnnotationKeyAlpha = "pod.alpha.kubernetes.io/init-containers"
	podInitContainersAnnotationKeyBeta  = "pod.beta.kubernetes.io/init-containers"
)

// createPodSpec creates a pod-spec for all tasks in a given pod.
func createPodSpec(tg *jobs.TaskGroup, pod pod, ctx generatorContext) (*k8s.PodSpec, map[string]string, error) {
	spec := &k8s.PodSpec{
		DNSPolicy:       "ClusterFirst",
		RestartPolicy:   RestartPolicyAlways,
		SecurityContext: &k8s.PodSecurityContext{},
	}

	// Volumes
	volumes, err := createVolumes(tg, pod, ctx)
	if err != nil {
		return nil, nil, maskAny(err)
	}
	spec.Volumes = volumes

	// Containers
	var allInitContainers []k8s.Container
	for _, t := range pod.tasks {
		if t.Network.IsHost() {
			spec.HostNetwork = true
		}
		initContainers, containers, extraVols, err := createTaskContainers(t, pod, ctx)
		if err != nil {
			return nil, nil, maskAny(err)
		}
		allInitContainers = append(allInitContainers, initContainers...)
		spec.Volumes = append(spec.Volumes, extraVols...)
		spec.Containers = append(spec.Containers, containers...)
	}

	annotations := make(map[string]string)
	if len(allInitContainers) > 0 {
		if len(spec.Containers) == 0 {
			// Take the last init container and use it as normal container
			spec.Containers = allInitContainers[len(allInitContainers)-1:]
			allInitContainers = allInitContainers[:len(allInitContainers)-1]
		}
		if len(allInitContainers) > 0 {
			raw, err := json.Marshal(allInitContainers)
			if err != nil {
				return nil, nil, maskAny(err)
			}
			annotations[podInitContainersAnnotationKeyAlpha] = string(raw)
			annotations[podInitContainersAnnotationKeyBeta] = string(raw)
		}
	}

	return spec, annotations, nil
}
