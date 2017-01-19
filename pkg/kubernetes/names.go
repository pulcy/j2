package kubernetes

import "strings"

const (
	// environment variable names
	EnvVarPodIP     = "J2_POD_IP"
	EnvVarPodName   = "J2_POD_NAME"
	EnvVarNodeName  = "J2_NODE_NAME"
	EnvVarNamespace = "J2_NAMESPACE"

	// Secret related
	EnvVarClusterID    = "CLUSTER_ID"
	EnvVarVaultAddress = "VAULT_ADDR"
	EnvVarVaultCACert  = "VAULT_CACERT"
	EnvVarVaultCAPath  = "VAULT_CAPATH"
)

const (
	// Well known secret names
	SecretClusterInfo = "j2-cluster-info"
	SecretVaultInfo   = "j2-vault-info"
)

const (
	// Load-balancer names
	LoadBalancerDNS = "lb-lb-srv.base"
)

const (
	// Docker registry secret field names
	DockerRegistrySecretDataFieldDockerConfigJSON = ".dockerconfigjson"
	DockerRegistrySecretTypeDockerConfigJSON      = "kubernetes.io/dockerconfigjson"
)

var (
	resourceNameReplacer = strings.NewReplacer(
		"/", "-",
		"_", "-",
	)
)

// ResourceName replaces all characters in the given name that are not valid for K8S resource names.
func ResourceName(fullName string) string {
	return strings.ToLower(resourceNameReplacer.Replace(fullName))
}
