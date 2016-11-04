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

package jobs

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/juju/errgo"

	"github.com/op/go-logging"
	"github.com/pulcy/j2/cluster"
	fg "github.com/pulcy/j2/flags"
	"github.com/pulcy/j2/pkg/vault"
)

const (
	privateLoadBalancerPort    = 81
	privateTcpLoadBalancerPort = 82
)

type jobFunctions struct {
	jobPath string
	options fg.Options
	cluster cluster.Cluster
	log     *logging.Logger
	vault.VaultConfig
	vault.GithubLoginData
}

// newJobFunctions creates a new instance of jobFunctions
func newJobFunctions(jobPath string, cluster cluster.Cluster, options fg.Options,
	log *logging.Logger, vaultConfig vault.VaultConfig, ghLoginData vault.GithubLoginData) *jobFunctions {
	absJobPath, _ := filepath.Abs(jobPath)
	return &jobFunctions{
		jobPath:         absJobPath,
		options:         options,
		cluster:         cluster,
		VaultConfig:     vaultConfig,
		GithubLoginData: ghLoginData,
		log:             log,
	}
}

// Functions returns all supported template functions
func (jf *jobFunctions) Functions() template.FuncMap {
	return template.FuncMap{
		"cat":          jf.cat,
		"env":          jf.getEnv,
		"opt":          jf.getOpt,
		"quote":        strconv.Quote,
		"replace":      strings.Replace,
		"trim":         strings.TrimSpace,
		"split":        strings.Split,
		"private_ipv4": func() string { return "${COREOS_PRIVATE_IPV4}" },
		"public_ipv4":  func() string { return "${COREOS_PUBLIC_IPV4}" },
		"hostname":     func() string { return "%H" },
		"machine_id":   func() string { return "%m" },
		"link_url":     linkURL,
		"link_tcp":     linkTCP,
		"link_tls":     linkTLS,
		"secret":       jf.vaultExtract,
		"include":      jf.include,
	}
}

func (jf *jobFunctions) Options() parseJobOptions {
	return parseJobOptions{
		Cluster: jf.cluster,
	}
}

// getEnv loads an environment value and returns an error if it is empty.
func (jf *jobFunctions) getEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", errgo.WithCausef(nil, ValidationError, "Missing environment variables '%s'", key)
	}
	return value, nil
}

// getOpt loads an option with given key and returns an error the option does not exist.
func (jf *jobFunctions) getOpt(key string) (string, error) {
	value, ok := jf.options.Get(key)
	if !ok {
		value, ok = jf.cluster.DefaultOptions.Get(key)
		if !ok {
			switch key {
			case "domain":
				return jf.cluster.Domain, nil
			case "stack":
				return jf.cluster.Stack, nil
			case "tunnel":
				return jf.cluster.Tunnel, nil
			case "instance-count":
				return strconv.Itoa(jf.cluster.InstanceCount), nil
			default:
				return "", errgo.WithCausef(nil, ValidationError, "Missing option '%s'", key)
			}
		}
	}
	if result, err := formatOptionValue(value, false); err != nil {
		return "", maskAny(err)
	} else {
		return result, nil
	}
}

func formatOptionValue(value interface{}, quote bool) (string, error) {
	if s, ok := value.(string); ok {
		if quote {
			return strconv.Quote(s), nil
		}
		return s, nil
	}
	if l, ok := value.([]interface{}); ok {
		var result []string
		for _, e := range l {
			fe, err := formatOptionValue(e, true)
			if err != nil {
				return "", maskAny(err)
			}
			result = append(result, fe)
		}
		return "[" + strings.Join(result, ", ") + "]", nil
	}
	if m, ok := value.(map[string]interface{}); ok {
		var result []string
		for k, v := range m {
			fv, err := formatOptionValue(v, true)
			if err != nil {
				return "", maskAny(err)
			}
			result = append(result, fmt.Sprintf("%s = %s", k, fv))
		}
		return "{\n" + strings.Join(result, "\n") + "}", nil
	}
	return "", maskAny(errgo.WithCausef(nil, ValidationError, "Unknown value type: %v", value))
}

// cat returns the content of a file.
func (jf *jobFunctions) cat(path string) (string, error) {
	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(jf.jobDir(), path)
	}
	raw, err := ioutil.ReadFile(absPath)
	if os.IsNotExist(err) {
		return "", errgo.WithCausef(nil, ValidationError, "File '%s' not found", absPath)
	} else if err != nil {
		return "", maskAny(err)
	}
	return string(raw), nil
}

// jobDir returns the folder containing the current jobPath.
func (jf *jobFunctions) jobDir() string {
	return filepath.Dir(jf.jobPath)
}

// linkURL creates an URL to the domain name (in private namespace) of the given link
func linkURL(linkName string) (string, error) {
	ln := LinkName(linkName)
	if err := ln.Validate(); err != nil {
		return "", maskAny(err)
	}
	return fmt.Sprintf("http://%s:%d", ln.PrivateDomainName(), privateLoadBalancerPort), nil
}

// linkTLS creates an URL with `tls` scheme to the domain name (in private TCP namespace) of the given link
func linkTLS(linkName string) (string, error) {
	ln := LinkName(linkName)
	if err := ln.Validate(); err != nil {
		return "", maskAny(err)
	}
	return fmt.Sprintf("tls://%s:%d", ln.PrivateDomainName(), privateTcpLoadBalancerPort), nil
}

// linkTCP creates a tcp URL to the domain name (in private TCP namespace) of the given link
func linkTCP(linkName string, port int) (string, error) {
	ln := LinkName(linkName)
	if err := ln.Validate(); err != nil {
		return "", maskAny(err)
	}
	return fmt.Sprintf("tcp://%s:%d", ln.PrivateDomainName(), port), nil
}

// vaultExtract extracts a value out of the current Vault.
func (jf *jobFunctions) vaultExtract(vaultPath string) (string, error) {
	vault, err := vault.NewVault(jf.VaultConfig, jf.log)
	if err != nil {
		return "", maskAny(err)
	}
	if err := vault.GithubLogin(jf.GithubLoginData); err != nil {
		return "", maskAny(err)
	}
	secret, err := vault.Extract(vaultPath, "value")
	if err != nil {
		return "", maskAny(err)
	}
	return secret, nil
}

func (jf *jobFunctions) include(name string) (string, error) {
	includeData, includePath, err := jf.readInclude(name)
	if err != nil {
		return "", maskAny(err)
	}

	// Create a template, add the function map, and parse the text.
	includeJF := *jf
	includeJF.jobPath = includePath
	tmpl, err := template.New("include-" + name).Funcs(includeJF.Functions()).Parse(string(includeData))
	if err != nil {
		return "", maskAny(err)
	}

	// Run the template to verify the output.
	buffer := &bytes.Buffer{}
	err = tmpl.Execute(buffer, includeJF.Options())
	if err != nil {
		return "", maskAny(err)
	}

	return buffer.String(), nil
}

func (jf *jobFunctions) readInclude(name string) ([]byte, string, error) {
	path := name
	if !filepath.IsAbs(name) {
		path = filepath.Join(filepath.Dir(jf.jobPath), name)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = path + ".hcl"
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, "", maskAny(err)
	}
	return data, path, nil
}
