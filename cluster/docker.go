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

// DockerOptions contains options used to generate docker commands for the jobs that run on this cluster.
type DockerOptions struct {
	// Arguments to add to the docker command for logging
	LoggingArgs []string `mapstructure:"docker-log-args,omitempty"`
	EnvFile     string   `mapstructure:"env-file,omitempty"`
}

// validate checks the values in the given cluster
func (do DockerOptions) validate() error {
	return nil
}
