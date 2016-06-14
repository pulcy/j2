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

type SecretList []Secret

// Validate checks the values of the given secret.
// If ok, return nil, otherwise returns an error.
func (list SecretList) Validate() error {
	for _, s := range list {
		if err := s.Validate(); err != nil {
			return maskAny(err)
		}
	}
	return nil
}

// AnyTargetEnviroment returns true if at least one of the secrets in the list
// has an environment variable as target.
func (list SecretList) AnyTargetEnviroment() bool {
	for _, s := range list {
		if ok, _ := s.TargetEnviroment(); ok {
			return true
		}
	}
	return false
}

// AnyTargetFile returns true if at least one of the secrets in the list
// has a file as target.
func (list SecretList) AnyTargetFile() bool {
	for _, s := range list {
		if ok, _ := s.TargetFile(); ok {
			return true
		}
	}
	return false
}
