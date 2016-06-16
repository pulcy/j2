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

package vault

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/go-rootcerts"
	"github.com/hashicorp/vault/api"
	"github.com/juju/errgo"
	"github.com/mitchellh/go-homedir"
	"github.com/op/go-logging"
)

type VaultConfig struct {
	VaultAddr   string
	VaultCACert string
	VaultCAPath string
}

type Vault struct {
	vaultClient *api.Client
	log         *logging.Logger
}

func NewVault(srvCfg VaultConfig, log *logging.Logger) (*Vault, error) {
	// Create a vault client
	config := api.DefaultConfig()
	if err := config.ReadEnvironment(); err != nil {
		return nil, maskAny(err)
	}
	var serverName string
	if srvCfg.VaultAddr != "" {
		log.Debugf("Setting vault address to %s", srvCfg.VaultAddr)
		config.Address = srvCfg.VaultAddr
		url, err := url.Parse(config.Address)
		if err != nil {
			return nil, maskAny(err)
		}
		host, _, err := net.SplitHostPort(url.Host)
		if err != nil {
			return nil, maskAny(err)
		}
		serverName = host
	}
	if srvCfg.VaultCACert != "" || srvCfg.VaultCAPath != "" {
		clientTLSConfig := config.HttpClient.Transport.(*http.Transport).TLSClientConfig
		if err := rootcerts.ConfigureTLS(clientTLSConfig, &rootcerts.Config{
			CAFile: srvCfg.VaultCACert,
			CAPath: srvCfg.VaultCAPath,
		}); err != nil {
			return nil, maskAny(err)
		}
		clientTLSConfig.ServerName = serverName
	}
	client, err := api.NewClient(config)
	if err != nil {
		return nil, maskAny(err)
	}

	return &Vault{
		log:         log,
		vaultClient: client,
	}, nil

}

type GithubLoginData struct {
	GithubToken     string
	GithubTokenPath string
	Mount           string // defaults to "github"
}

// GithubLogin performs a standard Github authentication and initializes the vaultClient with the resulting token.
func (s *Vault) GithubLogin(data GithubLoginData) error {
	// Read token
	var err error
	data.GithubToken, err = s.readGithubToken(data)
	if err != nil {
		return maskAny(err)
	}
	// Perform login
	s.vaultClient.ClearToken()
	logical := s.vaultClient.Logical()
	loginData := make(map[string]interface{})
	loginData["token"] = data.GithubToken
	if data.Mount == "" {
		data.Mount = "github"
	}
	path := fmt.Sprintf("auth/%s/login", data.Mount)
	if loginSecret, err := logical.Write(path, loginData); err != nil {
		return maskAny(err)
	} else if loginSecret.Auth == nil {
		return maskAny(errgo.WithCausef(nil, VaultError, "missing authentication in secret response"))
	} else {
		// Use token
		s.vaultClient.SetToken(loginSecret.Auth.ClientToken)
	}

	// We're done
	return nil
}

func (s *Vault) readGithubToken(data GithubLoginData) (string, error) {
	if data.GithubToken != "" {
		return data.GithubToken, nil
	}
	if data.GithubTokenPath == "" {
		return "", maskAny(errgo.WithCausef(nil, InvalidArgumentError, "No github token path set"))
	}
	path, err := homedir.Expand(data.GithubTokenPath)
	if err != nil {
		return "", maskAny(err)
	}
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return "", maskAny(err)
	}
	return strings.TrimSpace(string(raw)), nil
}

// extractSecret extracts a secret based on given variables
// Call a login method before calling this method.
func (s *Vault) Extract(secretPath, secretField string) (string, error) {
	if secretPath == "" {
		return "", maskAny(errgo.WithCausef(nil, InvalidArgumentError, "path not set"))
	}
	if secretField == "" {
		return "", maskAny(errgo.WithCausef(nil, InvalidArgumentError, "field not set"))
	}

	// Load secret
	s.log.Infof("Read %s#%s", secretPath, secretField)
	secret, err := s.vaultClient.Logical().Read(secretPath)
	if err != nil {
		return "", maskAny(errgo.WithCausef(nil, VaultError, "error reading %s: %s", secretPath, err))
	}
	if secret == nil {
		return "", maskAny(errgo.WithCausef(nil, VaultError, "no value found at %s", secretPath))
	}

	if value, ok := secret.Data[secretField]; !ok {
		return "", maskAny(errgo.WithCausef(nil, VaultError, "no field '%s' found at %s", secretField, secretPath))
	} else {
		return value.(string), nil
	}
}
