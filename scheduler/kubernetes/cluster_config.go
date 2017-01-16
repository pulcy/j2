// Copyright (c) 2016 Pulcy.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kubernetes

import (
	"fmt"

	"github.com/juju/errgo"

	k8s "github.com/YakLabs/k8s-client"
	pkg "github.com/pulcy/j2/pkg/kubernetes"
	"github.com/pulcy/j2/scheduler"
)

// ValidateCluster checks if the cluster is suitable to run the configured job.
func (s *k8sScheduler) ValidateCluster() error {
	// Check vault-info secret
	vaultInfo, err := s.client.GetSecret(s.defaultNamespace, pkg.SecretVaultInfo)
	if err != nil {
		return maskAny(errgo.Notef(err, "%s secret missing: %v", pkg.SecretVaultInfo, err))
	}
	if _, ok := vaultInfo.Data[pkg.EnvVarVaultAddress]; !ok {
		return maskAny(fmt.Errorf("%s secret missing data for %s", pkg.SecretVaultInfo, pkg.EnvVarVaultAddress))
	}
	_, caCertFound := vaultInfo.Data[pkg.EnvVarVaultCACert]
	_, caPathFound := vaultInfo.Data[pkg.EnvVarVaultCAPath]
	if !(caCertFound || caPathFound) {
		return maskAny(fmt.Errorf("%s secret missing data for %s or %s", pkg.SecretVaultInfo, pkg.EnvVarVaultCACert, pkg.EnvVarVaultCAPath))
	}
	// Check cluster-info secret
	clusterInfo, err := s.client.GetSecret(s.defaultNamespace, pkg.SecretClusterInfo)
	if err != nil {
		return maskAny(errgo.Notef(err, "%s secret missing: %v", pkg.SecretClusterInfo, err))
	}
	if _, ok := clusterInfo.Data[pkg.EnvVarClusterID]; !ok {
		return maskAny(fmt.Errorf("%s secret missing data for %s", pkg.SecretClusterInfo, pkg.EnvVarClusterID))
	}
	// Check registry pull secrets
	for _, secretName := range s.cluster.KubernetesOptions.RegistrySecrets {
		secret, err := s.client.GetSecret(s.defaultNamespace, secretName)
		if err != nil {
			return maskAny(errgo.Notef(err, "%s secret missing: %v", secretName, err))
		}
		if _, ok := secret.Data[pkg.DockerRegistrySecretDataFieldDockerConfigJSON]; !ok {
			return maskAny(fmt.Errorf("%s secret missing data for %s", secretName, pkg.DockerRegistrySecretDataFieldDockerConfigJSON))
		}
		if secret.Type != pkg.DockerRegistrySecretTypeDockerConfigJSON {
			return maskAny(fmt.Errorf("%s secret wrong type, got '%s' expected '%s'", secretName, secret.Type, pkg.DockerRegistrySecretTypeDockerConfigJSON))
		}
	}
	return nil
}

// ConfigureCluster configures the cluster for use by J2.
func (s *k8sScheduler) ConfigureCluster(config scheduler.ClusterConfig) error {
	// Fetch info (if needed)
	clusterID := config.ClusterID()
	registrySecretConfigs := make(map[string][]byte)
	if clusterID == "" {
		if s.defaultNamespace != "base" {
			// Try to fetch cluster info from base namespace
			secret, err := s.client.GetSecret("base", pkg.SecretClusterInfo)
			if err == nil {
				clusterID = string(secret.Data[pkg.EnvVarClusterID])
			}
			// Fetch data of registry secrets
			for _, secretName := range s.cluster.KubernetesOptions.RegistrySecrets {
				secret, err := s.client.GetSecret("base", secretName)
				if err == nil {
					config, ok := secret.Data[pkg.DockerRegistrySecretDataFieldDockerConfigJSON]
					if !ok {
						continue
					}
					if secret.Type != pkg.DockerRegistrySecretTypeDockerConfigJSON {
						continue
					}
					registrySecretConfigs[secretName] = config
				}
			}
		}
		if clusterID == "" {
			return maskAny(fmt.Errorf("clusterID cannot be empty"))
		}
	}
	// Ensure namespace exists
	if err := s.ensureNamespace(s.defaultNamespace); err != nil {
		return maskAny(err)
	}

	updateSecret := func(secretName string, values map[string]string, secretType k8s.SecretType) error {
		create := false
		secret, err := s.client.GetSecret(s.defaultNamespace, secretName)
		if err != nil {
			create = true
			secret = k8s.NewSecret(s.defaultNamespace, secretName)
		}
		if secretType != "" {
			secret.Type = secretType
		}
		for k, v := range values {
			raw := []byte(v)
			if len(v) == 0 {
				raw = []byte{}
			}
			secret.Data[k] = raw
		}
		if create {
			if _, err := s.client.CreateSecret(s.defaultNamespace, secret); err != nil {
				return maskAny(err)
			}
		} else {
			if _, err := s.client.UpdateSecret(s.defaultNamespace, secret); err != nil {
				return maskAny(err)
			}
		}
		return nil
	}
	// Update vault-info secret
	values := map[string]string{
		pkg.EnvVarVaultAddress: config.VaultAddress(),
		pkg.EnvVarVaultCACert:  config.VaultCACert(),
		pkg.EnvVarVaultCAPath:  config.VaultCAPath(),
	}
	if err := updateSecret(pkg.SecretVaultInfo, values, ""); err != nil {
		return maskAny(err)
	}

	// Update cluster-info secret
	if err := updateSecret(pkg.SecretClusterInfo, map[string]string{
		pkg.EnvVarClusterID: clusterID,
	}, ""); err != nil {
		return maskAny(err)
	}

	// Registry secrets
	for secretName, config := range registrySecretConfigs {
		values := map[string]string{
			pkg.DockerRegistrySecretDataFieldDockerConfigJSON: string(config),
		}
		if err := updateSecret(secretName, values, pkg.DockerRegistrySecretTypeDockerConfigJSON); err != nil {
			return maskAny(err)
		}
	}

	// Show cluster info
	nodes, err := s.client.ListNodes(nil)
	if err != nil {
		return maskAny(err)
	}
	for i, n := range nodes.Items {
		nodeInfo := n.Status.NodeInfo
		id := nodeInfo.MachineID
		if id == "" {
			id = nodeInfo.SystemUUID
		}
		fmt.Printf("Node %d: %s\n", i, id)
	}

	return nil
}
