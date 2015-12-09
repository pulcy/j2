package jobs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/juju/errgo"

	fg "arvika.pulcy.com/pulcy/deployit/flags"
)

const (
	privateLoadBalancerPort = 81
)

type jobFunctions struct {
	jobPath string
	options fg.Options
	cluster fg.Cluster
}

// newJobFunctions creates a new instance of jobFunctions
func newJobFunctions(jobPath string, cluster fg.Cluster, options fg.Options) *jobFunctions {
	absJobPath, _ := filepath.Abs(jobPath)
	return &jobFunctions{
		jobPath: absJobPath,
		options: options,
		cluster: cluster,
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
		"private_ipv4": jf.getPrivateIPV4,
		"public_ipv4":  jf.getPublicIPV4,
		"link_url":     jf.linkURL,
	}
}

// getPrivateIPV4 gets the COREOS_PRIVATE_IPV4 address.
func (jf *jobFunctions) getPrivateIPV4() string {
	return "${COREOS_PRIVATE_IPV4}"
}

// getPublicIPV4 gets the COREOS_PUBLIC_IPV4 address.
func (jf *jobFunctions) getPublicIPV4() string {
	return "${COREOS_PUBLIC_IPV4}"
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
		value, ok := jf.cluster.DefaultOptions.Get(key)
		if ok {
			return value, nil
		}
		switch key {
		case "domain":
			return jf.cluster.Domain, nil
		case "stack":
			return jf.cluster.Stack, nil
		case "tunnel":
			return jf.cluster.Tunnel, nil
		case "instance-count":
			return strconv.Itoa(jf.cluster.InstanceCount), nil
		}
		return "", errgo.WithCausef(nil, ValidationError, "Missing option '%s'", key)
	}
	return value, nil
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
func (jf *jobFunctions) linkURL(linkName string) (string, error) {
	ln := LinkName(linkName)
	if err := ln.Validate(); err != nil {
		return "", maskAny(err)
	}
	return fmt.Sprintf("http://%s:%d", ln.PrivateDomainName(), privateLoadBalancerPort), nil
}
