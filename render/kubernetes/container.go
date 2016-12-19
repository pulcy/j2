package kubernetes

import (
	"fmt"

	k8s "github.com/YakLabs/k8s-client"
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
func createTaskContainers(t *jobs.Task, pod pod, ctx generatorContext) ([]k8s.Container, []k8s.Container, error) {
	if t.Type.IsProxy() {
		// Proxy does not yield any containers
		return nil, nil, nil
	}
	c := &k8s.Container{
		Name:            resourceName(t.FullName(), ""),
		Image:           t.Image.String(),
		ImagePullPolicy: k8s.PullAlways,
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
		c.Env = append(c.Env, k8s.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	// Secrets that will be passed as environment variables
	var initContainers []k8s.Container
	var envSecrets []jobs.Secret
	for _, s := range t.Secrets {
		ok, key := s.TargetEnviroment()
		if !ok {
			continue
		}
		envSecrets = append(envSecrets, s)
		c.Env = append(c.Env, k8s.EnvVar{
			Name: key,
			ValueFrom: &k8s.EnvVarSource{
				SecretKeyRef: &k8s.SecretKeySelector{
					LocalObjectReference: k8s.LocalObjectReference{
						Name: taskSecretName(t),
					},
					Key: key,
				},
			},
		})
	}
	if len(envSecrets) > 0 {
		c, err := createSecretExtractionContainer(envSecrets, t, pod, ctx)
		if err != nil {
			return nil, nil, maskAny(err)
		}
		initContainers = append(initContainers, *c)
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
			mount := k8s.VolumeMount{
				Name:      createVolumeName(t, i),
				MountPath: v.Path,
			}
			mount.ReadOnly = v.IsReadOnly()
			c.VolumeMounts = append(c.VolumeMounts, mount)
		}
	}

	var containers []k8s.Container
	if t.Type.IsService() {
		containers = append(containers, *c)
	} else if t.Type.IsOneshot() {
		initContainers = append(initContainers, *c)
	}
	return initContainers, containers, nil
}
