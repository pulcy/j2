package kubernetes

import (
	"fmt"

	"k8s.io/client-go/pkg/api/v1"

	"github.com/pulcy/j2/jobs"
)

func createVolumeName(t *jobs.Task, volumeIndex int) string {
	return resourceName(t.FullName(), fmt.Sprintf("%s%d", kindVolume, volumeIndex))
}

type volumeTaskPair struct {
	Volume      jobs.Volume
	VolumeIndex int
	Task        *jobs.Task
}

// createVolumes creates the volumes defined in all tasks of a given pod.
func createVolumes(tg *jobs.TaskGroup, pod pod, ctx generatorContext) ([]v1.Volume, error) {
	// Collect all volumes
	var jVols []volumeTaskPair
	seenJVols := make(map[string]struct{})
	for _, t := range pod.tasks {
		for i, v := range t.Volumes {
			if _, ok := seenJVols[v.String()]; ok {
				continue
			}
			jVols = append(jVols, volumeTaskPair{Volume: v, VolumeIndex: i, Task: t})
			seenJVols[v.String()] = struct{}{}
		}
	}

	// Create volume for each
	var vols []v1.Volume
	for _, v := range jVols {
		vol := v1.Volume{
			Name: createVolumeName(v.Task, v.VolumeIndex),
		}
		if v.Volume.IsLocal() {
			vol.HostPath = &v1.HostPathVolumeSource{
				Path: v.Volume.HostPath,
			}
		} else {
			return nil, maskAny(fmt.Errorf("Non local volumes are not yet implemented"))
		}
		vols = append(vols, vol)
	}

	return vols, nil
}