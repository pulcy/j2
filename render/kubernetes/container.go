package kubernetes

import (
	"fmt"
	"sort"

	k8s "github.com/YakLabs/k8s-client"
	"github.com/pulcy/j2/jobs"
	pkg "github.com/pulcy/j2/pkg/kubernetes"
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
func createTaskContainers(t *jobs.Task, pod pod, ctx generatorContext, hostNetwork bool) ([]k8s.Container, []k8s.Container, []k8s.Volume, error) {
	if t.Type.IsProxy() {
		// Proxy does not yield any containers
		return nil, nil, nil, nil
	}
	c := newContainer(resourceName(t.FullName(), ""), t.Image.String())
	if len(t.Args) > 0 {
		c.Args = t.Args
	}

	// Exposed ports
	for _, p := range t.Ports {
		cp, err := createContainerPort(p, hostNetwork)
		if err != nil {
			return nil, nil, nil, maskAny(err)
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
	sort.Sort(envVarByName(c.Env))

	// Secrets that will be passed as environment variables or as file
	var initContainers []k8s.Container
	var vols []k8s.Volume
	var envSecrets []jobs.Secret
	var fileSecrets []jobs.Secret
	for _, s := range t.Secrets {
		if ok, key := s.TargetEnviroment(); ok {
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
		} else if ok, _ := s.TargetFile(); ok {
			fileSecrets = append(fileSecrets, s)
		}
	}
	if len(envSecrets) > 0 {
		c, secVols, err := createSecretEnvVarExtractionContainer(envSecrets, t, pod, ctx)
		if err != nil {
			return nil, nil, nil, maskAny(err)
		}
		initContainers = append(initContainers, *c)
		vols = append(vols, secVols...)
	}
	if len(fileSecrets) > 0 {
		cs, secVols, secVolMounts, err := createSecretFileExtractionContainers(fileSecrets, t, pod, ctx)
		if err != nil {
			return nil, nil, nil, maskAny(err)
		}
		initContainers = append(initContainers, cs...)
		vols = append(vols, secVols...)
		c.VolumeMounts = append(c.VolumeMounts, secVolMounts...)
	}

	// J2 specific Environment variables
	c.Env = append(c.Env,
		createEnvVarFromField(pkg.EnvVarPodIP, "status.podIP"),
		createEnvVarFromField(pkg.EnvVarPodName, "metadata.name"),
		createEnvVarFromField(pkg.EnvVarNamespace, "metadata.namespace"),
		createEnvVarFromField(pkg.EnvVarNodeName, "spec.nodeName"),
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
					return nil, nil, nil, maskAny(fmt.Errorf("Task '%s' not found in VolumesFrom of '%s'", name, current.Name))
				}
				mountTasks = append(mountTasks, pod.tasks[otherIndex])
			}
		}
	}
	// Create mounts
	for _, t := range mountTasks {
		for _, v := range t.Volumes {
			volName, err := createVolumeName(t.GroupName(), v)
			if err != nil {
				return nil, nil, nil, maskAny(err)
			}
			mount := k8s.VolumeMount{
				Name:      volName,
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
	return initContainers, containers, vols, nil
}

func newContainer(name, image string) *k8s.Container {
	return &k8s.Container{
		Name:                   name,
		Image:                  image,
		ImagePullPolicy:        k8s.PullAlways,
		TerminationMessagePath: "/dev/termination-log",
		Resources:              &k8s.ResourceRequirements{},
	}
}

type envVarByName []k8s.EnvVar

func (l envVarByName) Len() int           { return len(l) }
func (l envVarByName) Less(i, j int) bool { return l[i].Name < l[j].Name }
func (l envVarByName) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
