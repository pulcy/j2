package kubernetes

import "strings"

const (
	// environment variable names
	EnvVarPodIP    = "J2_POD_IP"
	EnvVarNodeName = "J2_NODE_NAME"

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

var (
	resourceNameReplacer = strings.NewReplacer(
		"/", "-",
		"_", "-",
	)
)

// ResourceName replaces all characters in the given name that are not valid for K8S resource names.
func ResourceName(fullName string) string {
	return resourceNameReplacer.Replace(fullName)
}
