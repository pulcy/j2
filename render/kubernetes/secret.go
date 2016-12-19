package kubernetes

import (
	"fmt"

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

// createSecretExtractionContainer creates an init-container that uses vault-monkey to extract one or more environment
// secrets into a kubernetes secret.
func createSecretExtractionContainer(secrets []jobs.Secret, t *jobs.Task, pod pod, ctx generatorContext) (*k8s.Container, []k8s.Volume, error) {
	caCertPath := "/etc/vault/vault.crt"
	vaultInfoVolumeName := "vault-info"
	jobID := t.JobID()
	if jobID == "" {
		return nil, nil, maskAny(fmt.Errorf("Job has no ID which is required for secrets in task %s", t.FullName()))
	}
	args := []string{
		"extract",
		"env",
		"--job-id", jobID,
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
	)

	return c, []k8s.Volume{vaultInfoVol}, nil
}
