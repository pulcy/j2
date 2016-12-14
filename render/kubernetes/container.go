package kubernetes

import (
	"fmt"

	"github.com/pulcy/j2/jobs"
	"k8s.io/client-go/pkg/api/v1"
)

// createTaskContainers returns the init-containers and containers needed for the given task.
func createTaskContainers(t *jobs.Task, pod pod, ctx generatorContext) ([]v1.Container, []v1.Container, error) {
	if t.Type.IsProxy() {
		// Proxy does not yield any containers
		return nil, nil, nil
	}
	c := v1.Container{
		Name:            resourceName(t.FullName(), ""),
		Image:           t.Image.String(),
		ImagePullPolicy: v1.PullAlways,
		Args:            t.Args,
	}

	// Exposed ports
	for _, p := range t.Ports {
		cp, err := createContainerPort(p)
		if err != nil {
			return nil, nil, maskAny(err)
		}
		c.Ports = append(c.Ports, cp)
	}

	// Environment variables
	for k, v := range t.Environment {
		c.Env = append(c.Env, v1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	// Mount volumes
	// First find all tasks to mount volumes from
	mountTasks := jobs.TaskList{t}
	for i := 0; i < len(mountTasks); i++ {
		current := mountTasks[i]
		for _, name := range current.VolumesFrom {
			if mountTasks.IndexByName(name) < 0 {
				otherIndex := pod.tasks.IndexByName(name)
				if otherIndex < 0 {
					return nil, nil, maskAny(fmt.Errorf("Task '%s' not found in VolumesFrom of '%s'", name, current.Name))
				}
				mountTasks = append(mountTasks, pod.tasks[otherIndex])
			}
		}
	}
	// Create mounts
	for _, t := range mountTasks {
		for i, v := range t.Volumes {
			mount := v1.VolumeMount{
				Name:      createVolumeName(t, i),
				MountPath: v.Path,
			}
			mount.ReadOnly = v.IsReadOnly()
			c.VolumeMounts = append(c.VolumeMounts, mount)
		}
	}

	var initContainers, containers []v1.Container
	if t.Type.IsService() {
		containers = append(containers, c)
	} else if t.Type.IsOneshot() {
		initContainers = append(initContainers, c)
	}
	return initContainers, containers, nil
}
