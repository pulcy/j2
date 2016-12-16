package kubernetes

import (
	"fmt"

	"github.com/ericchiang/k8s"
	"github.com/ericchiang/k8s/api/v1"
	"github.com/pulcy/j2/jobs"
)

const (
	// PullAlways means that kubelet always attempts to pull the latest image.  Container will fail If the pull fails.
	PullAlways = "Always"
	// PullNever means that kubelet never pulls an image, but only uses a local image.  Container will fail if the image isn't present
	PullNever = "Never"
	// PullIfNotPresent means that kubelet pulls if the image isn't present on disk. Container will fail if the image isn't present and the pull fails.
	PullIfNotPresent = "IfNotPresent"
)

// createTaskContainers returns the init-containers and containers needed for the given task.
func createTaskContainers(t *jobs.Task, pod pod, ctx generatorContext) ([]*v1.Container, []*v1.Container, error) {
	if t.Type.IsProxy() {
		// Proxy does not yield any containers
		return nil, nil, nil
	}
	c := &v1.Container{
		Name:            k8s.StringP(resourceName(t.FullName(), "")),
		Image:           k8s.StringP(t.Image.String()),
		ImagePullPolicy: k8s.StringP(PullAlways),
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
		c.Env = append(c.Env, &v1.EnvVar{
			Name:  k8s.StringP(k),
			Value: k8s.StringP(v),
		})
	}

	// J2 specific Environment variables
	c.Env = append(c.Env,
		createEnvVarFromField(envVarPodIP, "status.podIP"),
		createEnvVarFromField(envVarNodeName, "spec.nodeName"),
	)

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
			mount := &v1.VolumeMount{
				Name:      k8s.StringP(createVolumeName(t, i)),
				MountPath: k8s.StringP(v.Path),
			}
			mount.ReadOnly = k8s.BoolP(v.IsReadOnly())
			c.VolumeMounts = append(c.VolumeMounts, mount)
		}
	}

	var initContainers, containers []*v1.Container
	if t.Type.IsService() {
		containers = append(containers, c)
	} else if t.Type.IsOneshot() {
		initContainers = append(initContainers, c)
	}
	return initContainers, containers, nil
}

// createEnvVarFromField creates a v1.EnvVar with a ValueFrom set to a ObjectFieldSelector
func createEnvVarFromField(key, fieldPath string) *v1.EnvVar {
	return &v1.EnvVar{
		Name: k8s.StringP(key),
		ValueFrom: &v1.EnvVarSource{
			FieldRef: &v1.ObjectFieldSelector{
				FieldPath: k8s.StringP(fieldPath),
			},
		},
	}
}
