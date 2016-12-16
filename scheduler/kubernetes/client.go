package kubernetes

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ericchiang/k8s"
)

func createClientFromConfig(kubeConfig string) (*k8s.Client, error) {
	// Load configuration
	config, err := loadKubeConfig(kubeConfig)
	if err != nil {
		return nil, maskAny(err)
	}
	context, err := config.GetCurrentContext()
	if err != nil {
		return nil, maskAny(err)
	}
	cluster, err := config.GetCluster(context.Cluster)
	if err != nil {
		return nil, maskAny(err)
	}
	user, err := config.GetUser(context.AuthInfo)
	if err != nil {
		return nil, maskAny(err)
	}

	// Load client cert.
	clientCert, err := tls.LoadX509KeyPair(user.ClientCertificate, user.ClientKey)
	if err != nil {
		return nil, maskAny(err)
	}

	// Load API server's CA.
	caData, err := ioutil.ReadFile(cluster.CertificateAuthority)
	if err != nil {
		return nil, maskAny(err)
	}
	rootCAs := x509.NewCertPool()
	if !rootCAs.AppendCertsFromPEM(caData) {
		return nil, maskAny(fmt.Errorf("Failed to append CA certificates"))
	}

	// Create a client with a custom TLS config.
	client := &k8s.Client{
		Endpoint:  cluster.Server,
		Namespace: "default",
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:      rootCAs,
					Certificates: []tls.Certificate{clientCert},
				},
			},
		},
	}
	return client, nil
}
