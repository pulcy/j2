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

package cluster

import (
	homedir "github.com/mitchellh/go-homedir"
)

// KubernetesOptions contains options used to generate kubernetes resources for the jobs that run on this cluster.
type KubernetesOptions struct {
	KubeConfig                string   // Full path of the kube-config file
	Context                   string   `mapstructure:"context,omitempty"`          // Name of the context (defined in KubeConfig) to use
	RegistrySecrets           []string `mapstructure:"registry-secrets,omitempty"` // Name of secrets added to imagePullSecrets of each generated pod.
	GlobalInstanceConstraints []string `mapstructure:"global-instance-constraints,omitempty"`
	Domain                    string   `mapstructure:"domain,omitempty"` // CLuster local domain (defaults to 'cluster.local')
}

// validate checks the values in the given cluster
func (o KubernetesOptions) validate() error {
	return nil
}

func (o *KubernetesOptions) setDefaults() {
	if len(o.GlobalInstanceConstraints) == 0 {
		o.GlobalInstanceConstraints = []string{
			"odd=true",
			"even=true",
		}
	}
	if o.KubeConfig == "" {
		path, _ := homedir.Expand("~/.kube/config")
		o.KubeConfig = path
	}
	if o.Domain == "" {
		o.Domain = "cluster.local"
	}
}
