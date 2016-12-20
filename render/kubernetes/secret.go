package kubernetes

import (
	"fmt"
	"path"

	"path/filepath"

	k8s "github.com/YakLabs/k8s-client"
	"github.com/pulcy/j2/jobs"
	pkg "github.com/pulcy/j2/pkg/kubernetes"
)

// createSecrets create a secret for every task that uses one or more secrets.
func createSecrets(tg *jobs.TaskGroup, pod pod, ctx generatorContext) ([]k8s.Secret, error) {
	var secrets []k8s.Secret
	for _, t := range pod.tasks {
		if len(t.Secrets) == 0 {
			continue
		}
		d := k8s.NewSecret(ctx.Namespace, taskSecretName(t))
		setTaskGroupLabelsAnnotations(&d.ObjectMeta, tg)
		secrets = append(secrets, *d)
	}

	return secrets, nil
}

// createSecretEnvVarExtractionContainer creates an init-container that uses vault-monkey to extract one or more environment
// secrets into a kubernetes secret.
func createSecretEnvVarExtractionContainer(secrets []jobs.Secret, t *jobs.Task, pod pod, ctx generatorContext) (*k8s.Container, []k8s.Volume, error) {
	caCertPath := "/etc/vault/vault.crt"
	vaultInfoVolumeName := "vault-info-env"
	jobID := t.JobID()
	if jobID == "" {
		return nil, nil, maskAny(fmt.Errorf("Job has no ID which is required for secrets in task %s", t.FullName()))
	}
	args := []string{
		"extract",
		"env",
		"--job-id", jobID,
		"--kubernetes-pod-name", fmt.Sprintf("$(%s)", pkg.EnvVarNodeName),
		"--kubernetes-pod-ip", fmt.Sprintf("$(%s)", pkg.EnvVarPodIP),
		"--kubernetes-cluster-info-secret-name", pkg.SecretClusterInfo,
		"--kubernetes-secret-name", taskSecretName(t),
		"--vault-cacert", caCertPath,
	}
	for _, s := range secrets {
		arg := fmt.Sprintf("%s=%v", s.Environment, s.VaultPath())
		args = append(args, arg)
	}
	c := &k8s.Container{
		Name:            resourceName(t.FullName(), "-vm"),
		Image:           ctx.ImageVaultMonkey,
		ImagePullPolicy: k8s.PullIfNotPresent,
		Args:            args,
	}

	// Volumes
	vaultInfoVol := k8s.Volume{
		Name: vaultInfoVolumeName,
		VolumeSource: k8s.VolumeSource{
			Secret: &k8s.SecretVolumeSource{
				SecretName: pkg.SecretVaultInfo,
				Items: []k8s.KeyToPath{
					k8s.KeyToPath{
						Key:  pkg.EnvVarVaultCACert,
						Path: filepath.Base(caCertPath),
					},
				},
			},
		},
	}

	// Volume mounts
	c.VolumeMounts = append(c.VolumeMounts, k8s.VolumeMount{
		Name:      vaultInfoVolumeName,
		ReadOnly:  true,
		MountPath: filepath.Dir(caCertPath),
	})

	// Environment variables
	c.Env = append(c.Env,
		createEnvVarFromSecret(pkg.EnvVarVaultAddress, pkg.SecretVaultInfo, pkg.EnvVarVaultAddress),
		createEnvVarFromField(pkg.EnvVarNodeName, "spec.nodeName"),
		createEnvVarFromField(pkg.EnvVarPodIP, "status.podIP"),
	)

	return c, []k8s.Volume{vaultInfoVol}, nil
}

// createSecretFileExtractionContainers creates a init-containers that use vault-monkey to extract file secrets into a memory backend volume.
func createSecretFileExtractionContainers(secrets []jobs.Secret, t *jobs.Task, pod pod, ctx generatorContext) ([]k8s.Container, []k8s.Volume, []k8s.VolumeMount, error) {
	caCertPath := "/etc/vault/vault.crt"
	vaultInfoVolumeName := "vault-info-file"
	jobID := t.JobID()
	if jobID == "" {
		return nil, nil, nil, maskAny(fmt.Errorf("Job has no ID which is required for secrets in task %s", t.FullName()))
	}
	// All secrets must be extract to the same folder
	folder := ""
	for _, s := range secrets {
		_, sPath := s.TargetFile()
		sFolder := path.Dir(sPath)
		if folder == "" {
			folder = sFolder
		} else if sFolder != folder {
			return nil, nil, nil, maskAny(fmt.Errorf("All file secrets on task '%s' must have same folder.", t.FullName()))
		}
	}
	switch folder {
	case "", ".", "/":
		return nil, nil, nil, maskAny(fmt.Errorf("Invalid root folder for file secrets '%s' in task '%s'.", folder, t.FullName()))
	}

	//panic("folder='" + folder + "'")

	secretBackingVolumeName := "vault-files"
	secretBackingVolume := k8s.Volume{
		Name: secretBackingVolumeName,
		VolumeSource: k8s.VolumeSource{
			EmptyDir: &k8s.EmptyDirVolumeSource{
				Medium: k8s.StorageMedium("Memory"),
			},
		},
	}
	secretBackingVolumeMountRO := k8s.VolumeMount{
		Name:      secretBackingVolumeName,
		ReadOnly:  true,
		MountPath: folder,
	}
	secretBackingVolumeMountRW := k8s.VolumeMount{
		Name:      secretBackingVolumeName,
		ReadOnly:  false,
		MountPath: folder,
	}

	vaultInfoVol := k8s.Volume{
		Name: vaultInfoVolumeName,
		VolumeSource: k8s.VolumeSource{
			Secret: &k8s.SecretVolumeSource{
				SecretName: pkg.SecretVaultInfo,
				Items: []k8s.KeyToPath{
					k8s.KeyToPath{
						Key:  pkg.EnvVarVaultCACert,
						Path: filepath.Base(caCertPath),
					},
				},
			},
		},
	}

	var containers []k8s.Container
	for i, s := range secrets {
		_, path := s.TargetFile()
		args := []string{
			"extract",
			"file",
			"--job-id", jobID,
			"--kubernetes-pod-name", fmt.Sprintf("$(%s)", pkg.EnvVarNodeName),
			"--kubernetes-pod-ip", fmt.Sprintf("$(%s)", pkg.EnvVarPodIP),
			"--kubernetes-cluster-info-secret-name", pkg.SecretClusterInfo,
			"--vault-cacert", caCertPath,
			"--target", path,
			s.VaultPath(),
		}
		c := &k8s.Container{
			Name:            resourceName(t.FullName(), fmt.Sprintf("-vmf%d", i)),
			Image:           ctx.ImageVaultMonkey,
			ImagePullPolicy: k8s.PullIfNotPresent,
			Args:            args,
		}

		// Volume mounts
		c.VolumeMounts = append(c.VolumeMounts,
			secretBackingVolumeMountRW,
			k8s.VolumeMount{
				Name:      vaultInfoVolumeName,
				ReadOnly:  true,
				MountPath: filepath.Dir(caCertPath),
			})

		// Environment variables
		c.Env = append(c.Env,
			createEnvVarFromSecret(pkg.EnvVarVaultAddress, pkg.SecretVaultInfo, pkg.EnvVarVaultAddress),
			createEnvVarFromField(pkg.EnvVarNodeName, "spec.nodeName"),
			createEnvVarFromField(pkg.EnvVarPodIP, "status.podIP"),
		)
		containers = append(containers, *c)
	}

	return containers, []k8s.Volume{secretBackingVolume, vaultInfoVol}, []k8s.VolumeMount{secretBackingVolumeMountRO}, nil
}
