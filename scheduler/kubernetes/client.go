package kubernetes

import (
	k8s "github.com/YakLabs/k8s-client"
	"github.com/YakLabs/k8s-client/http"
)

func createClientFromConfig(kubeConfig, contextName string) (k8s.Client, error) {
	// Load configuration
	config, err := loadKubeConfig(kubeConfig)
	if err != nil {
		return nil, maskAny(err)
	}
	var context *Context
	if contextName == "" {
		context, err = config.GetCurrentContext()
		if err != nil {
			return nil, maskAny(err)
		}
	} else {
		context, err = config.GetContext(contextName)
		if err != nil {
			return nil, maskAny(err)
		}
	}
	cluster, err := config.GetCluster(context.Cluster)
	if err != nil {
		return nil, maskAny(err)
	}
	user, err := config.GetUser(context.AuthInfo)
	if err != nil {
		return nil, maskAny(err)
	}

	// Prepare options
	opts := []http.OptionsFunc{
		// Endpoint
		http.SetServer(cluster.Server),
	}

	// Load client cert.
	if len(user.ClientCertificateData) > 0 {
		opts = append(opts, http.SetClientCert(user.ClientCertificateData))
	} else {
		opts = append(opts, http.SetClientCertFromFile(user.ClientCertificate))
	}
	if len(user.ClientKeyData) > 0 {
		opts = append(opts, http.SetClientKey(user.ClientKeyData))
	} else {
		opts = append(opts, http.SetClientKeyFromFile(user.ClientKey))
	}

	// API server's CA.
	if len(cluster.CertificateAuthorityData) > 0 {
		opts = append(opts, http.SetCA(cluster.CertificateAuthorityData))
	} else {
		opts = append(opts, http.SetCAFromFile(cluster.CertificateAuthority))
	}
	if cluster.InsecureSkipTLSVerify {
		opts = append(opts, http.SetInsecureSkipVerify(true))
	}

	client, err := http.New(opts...)
	if err != nil {
		return nil, maskAny(err)
	}
	return client, nil
}
