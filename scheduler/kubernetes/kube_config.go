package kubernetes

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/ghodss/yaml"
)

func loadKubeConfig(kubeConfig string) (*Config, error) {
	yamlData, err := ioutil.ReadFile(kubeConfig)
	if err != nil {
		return nil, maskAny(err)
	}
	jsonData, err := yaml.YAMLToJSON(yamlData)
	if err != nil {
		return nil, maskAny(err)
	}
	config := Config{}
	if err := json.Unmarshal(jsonData, &config); err != nil {
		return nil, maskAny(err)
	}
	return &config, nil
}

// Where possible, json tags match the cli argument names.
// Top level config objects and all values required for proper functioning are not "omitempty".  Any truly optional piece of config is allowed to be omitted.

// Config holds the information needed to build connect to remote kubernetes clusters as a given user
// IMPORTANT if you add fields to this struct, please update IsConfigEmpty()
type Config struct {
	// Legacy field from pkg/api/types.go TypeMeta.
	// TODO(jlowdermilk): remove this after eliminating downstream dependencies.
	// +optional
	Kind string `json:"kind,omitempty"`
	// DEPRECATED: APIVersion is the preferred api version for communicating with the kubernetes cluster (v1, v2, etc).
	// Because a cluster can run multiple API groups and potentially multiple versions of each, it no longer makes sense to specify
	// a single value for the cluster version.
	// This field isn't really needed anyway, so we are deprecating it without replacement.
	// It will be ignored if it is present.
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
	// Preferences holds general information to be use for cli interactions
	Preferences Preferences `json:"preferences"`
	// Clusters is a map of referencable names to cluster configs
	Clusters []ClusterEntry `json:"clusters"`
	// AuthInfos is a map of referencable names to user configs
	AuthInfos []AuthInfoEntry `json:"users"`
	// Contexts is a map of referencable names to context configs
	Contexts []ContextEntry `json:"contexts"`
	// CurrentContext is the name of the context that you would like to use by default
	CurrentContext string `json:"current-context"`
	// Extensions holds additional information. This is useful for extenders so that reads and writes don't clobber unknown fields
	// +optional
	//Extensions map[string]runtime.Object `json:"extensions,omitempty"`
}

func (c *Config) GetCurrentContext() (*Context, error) {
	if c.CurrentContext == "" {
		return nil, maskAny(fmt.Errorf("No current-context specified"))
	}
	for _, ctx := range c.Contexts {
		if ctx.Name == c.CurrentContext {
			return ctx.Context, nil
		}
	}
	return nil, maskAny(fmt.Errorf("Context '%s' not found", c.CurrentContext))
}

func (c *Config) GetCluster(name string) (*Cluster, error) {
	for _, c := range c.Clusters {
		if c.Name == name {
			return c.Cluster, nil
		}
	}
	return nil, maskAny(fmt.Errorf("Cluster '%s' not found", name))
}

func (c *Config) GetUser(name string) (*AuthInfo, error) {
	for _, c := range c.AuthInfos {
		if c.Name == name {
			return c.AuthInfo, nil
		}
	}
	return nil, maskAny(fmt.Errorf("User '%s' not found", name))
}

// IMPORTANT if you add fields to this struct, please update IsConfigEmpty()
type Preferences struct {
	// +optional
	Colors bool `json:"colors,omitempty"`
	// Extensions holds additional information. This is useful for extenders so that reads and writes don't clobber unknown fields
	// +optional
	//Extensions map[string]runtime.Object `json:"extensions,omitempty"`
}

type ClusterEntry struct {
	Name    string   `json:"name"`
	Cluster *Cluster `json:"cluster"`
}

// Cluster contains information about how to communicate with a kubernetes cluster
type Cluster struct {
	// LocationOfOrigin indicates where this object came from.  It is used for round tripping config post-merge, but never serialized.
	LocationOfOrigin string
	// Server is the address of the kubernetes cluster (https://hostname:port).
	Server string `json:"server"`
	// APIVersion is the preferred api version for communicating with the kubernetes cluster (v1, v2, etc).
	// +optional
	APIVersion string `json:"api-version,omitempty"`
	// InsecureSkipTLSVerify skips the validity check for the server's certificate. This will make your HTTPS connections insecure.
	// +optional
	InsecureSkipTLSVerify bool `json:"insecure-skip-tls-verify,omitempty"`
	// CertificateAuthority is the path to a cert file for the certificate authority.
	// +optional
	CertificateAuthority string `json:"certificate-authority,omitempty"`
	// CertificateAuthorityData contains PEM-encoded certificate authority certificates. Overrides CertificateAuthority
	// +optional
	CertificateAuthorityData []byte `json:"certificate-authority-data,omitempty"`
	// Extensions holds additional information. This is useful for extenders so that reads and writes don't clobber unknown fields
	// +optional
	//Extensions map[string]runtime.Object `json:"extensions,omitempty"`
}

type AuthInfoEntry struct {
	Name     string    `json:"name"`
	AuthInfo *AuthInfo `json:"user"`
}

// AuthInfo contains information that describes identity information.  This is use to tell the kubernetes cluster who you are.
type AuthInfo struct {
	// LocationOfOrigin indicates where this object came from.  It is used for round tripping config post-merge, but never serialized.
	LocationOfOrigin string
	// ClientCertificate is the path to a client cert file for TLS.
	// +optional
	ClientCertificate string `json:"client-certificate,omitempty"`
	// ClientCertificateData contains PEM-encoded data from a client cert file for TLS. Overrides ClientCertificate
	// +optional
	ClientCertificateData []byte `json:"client-certificate-data,omitempty"`
	// ClientKey is the path to a client key file for TLS.
	// +optional
	ClientKey string `json:"client-key,omitempty"`
	// ClientKeyData contains PEM-encoded data from a client key file for TLS. Overrides ClientKey
	// +optional
	ClientKeyData []byte `json:"client-key-data,omitempty"`
	// Token is the bearer token for authentication to the kubernetes cluster.
	// +optional
	Token string `json:"token,omitempty"`
	// TokenFile is a pointer to a file that contains a bearer token (as described above).  If both Token and TokenFile are present, Token takes precedence.
	// +optional
	TokenFile string `json:"tokenFile,omitempty"`
	// Impersonate is the username to act-as.
	// +optional
	Impersonate string `json:"act-as,omitempty"`
	// Username is the username for basic authentication to the kubernetes cluster.
	// +optional
	Username string `json:"username,omitempty"`
	// Password is the password for basic authentication to the kubernetes cluster.
	// +optional
	Password string `json:"password,omitempty"`
	// AuthProvider specifies a custom authentication plugin for the kubernetes cluster.
	// +optional
	AuthProvider *AuthProviderConfig `json:"auth-provider,omitempty"`
	// Extensions holds additional information. This is useful for extenders so that reads and writes don't clobber unknown fields
	// +optional
	//Extensions map[string]runtime.Object `json:"extensions,omitempty"`
}

type ContextEntry struct {
	Name    string   `json:"name"`
	Context *Context `json:"context"`
}

// Context is a tuple of references to a cluster (how do I communicate with a kubernetes cluster), a user (how do I identify myself), and a namespace (what subset of resources do I want to work with)
type Context struct {
	Name string `json:"name"`
	// LocationOfOrigin indicates where this object came from.  It is used for round tripping config post-merge, but never serialized.
	LocationOfOrigin string
	// Cluster is the name of the cluster for this context
	Cluster string `json:"cluster"`
	// AuthInfo is the name of the authInfo for this context
	AuthInfo string `json:"user"`
	// Namespace is the default namespace to use on unspecified requests
	// +optional
	Namespace string `json:"namespace,omitempty"`
	// Extensions holds additional information. This is useful for extenders so that reads and writes don't clobber unknown fields
	// +optional
	//Extensions map[string]runtime.Object `json:"extensions,omitempty"`
}

// AuthProviderConfig holds the configuration for a specified auth provider.
type AuthProviderConfig struct {
	Name string `json:"name"`
	// +optional
	Config map[string]string `json:"config,omitempty"`
}
