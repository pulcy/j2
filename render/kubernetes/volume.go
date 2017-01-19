package kubernetes

import (
	"crypto/sha256"
	"fmt"

	k8s "github.com/YakLabs/k8s-client"
	"github.com/pulcy/j2/jobs"
)

func createVolumeName(tgName jobs.TaskGroupName, v jobs.Volume) (string, error) {
	var suffix string
	switch v.Type {
	case jobs.VolumeTypeLocal:
		// Create suffix as hash of HostPath
		hash := sha256.Sum256([]byte(v.HostPath))
		suffix = fmt.Sprintf("%x", hash[:8])
	default:
		// Unsupport type
		return "", maskAny(fmt.Errorf("Non local volumes are not yet implemented"))
	}
	return resourceName(tgName.String(), fmt.Sprintf("%s-%s", kindVolume, suffix)), nil
}

func createVolumeForSecretsName(t *jobs.Task) string {
	return resourceName(t.FullName(), fmt.Sprintf("%s-sec", kindVolume))
}

// createVolumes creates the volumes defined in all tasks of a given pod.
func createVolumes(tg *jobs.TaskGroup, pod pod, ctx generatorContext) ([]k8s.Volume, error) {
	// Create volume for each J2 volume.
	// When adding them to a list, duplicate volumes will be filtered out.
	var vols []k8s.Volume
	for _, t := range pod.tasks {
		for _, v := range t.Volumes {
			volName, err := createVolumeName(tg.Name, v)
			if err != nil {
				return nil, maskAny(err)
			}
			vol := k8s.Volume{
				Name: volName,
			}
			if v.IsLocal() {
				vol.VolumeSource.HostPath = &k8s.HostPathVolumeSource{
					Path: v.HostPath,
				}
			} else {
				return nil, maskAny(fmt.Errorf("Non local volumes are not yet implemented"))
			}
			vols = appendVolumes(vols, vol)
		}
	}

	return vols, nil
}

// appendVolumes adds volumes, skipping duplicate volumes
func appendVolumes(list []k8s.Volume, toAdd ...k8s.Volume) []k8s.Volume {
	if len(toAdd) == 0 {
		return list
	}
	existingNames := make(map[string]struct{})
	for _, v := range list {
		existingNames[v.Name] = struct{}{}
	}
	for _, v := range toAdd {
		if _, found := existingNames[v.Name]; found {
			// Skip duplicate names
			fmt.Printf("Skipping duplicate volume '%s'\n", v.Name)
		} else {
			list = append(list, v)
			existingNames[v.Name] = struct{}{}
		}
	}
	return list
}
