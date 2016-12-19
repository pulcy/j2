package kubernetes

import (
	"fmt"

	k8s "github.com/YakLabs/k8s-client"
	"github.com/pulcy/j2/jobs"
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
func createSecretExtractionContainer(secrets []jobs.Secret, t *jobs.Task, pod pod, ctx generatorContext) (*k8s.Container, error) {
	args := []string{
		"extract",
		"env",
		"--job-id", t.JobID(),
		"--kubernetes-cluster-info-secret-name", secretClusterInfo,
		"--kubernetes-secret-name", taskSecretName(t),
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

	// Environment variables
	c.Env = append(c.Env,
		createEnvVarFromSecret(envVarVaultAddress, secretVaultInfo, envVarVaultAddress),
		createEnvVarFromSecret(envVarVaultCACert, secretVaultInfo, envVarVaultCACert),
		createEnvVarFromSecret(envVarVaultCAPath, secretVaultInfo, envVarVaultCAPath),
	)

	return c, nil
}
