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

// Metrics contains a specification of a metrics provides by a task.
type Metrics struct {
	Port      int    `json:"port,omitempty" mapstructure:"port,omitempty"`
	Path      string `json:"path,omitempty" mapstructure:"path,omitempty"`
	RulesPath string `json:"rules-path,omitempty" mapstructure:"rules-path,omitempty"`
}

func (m Metrics) replaceVariables(ctx *variableContext) Metrics {
	m.Path = ctx.replaceString(m.Path)
	m.RulesPath = ctx.replaceString(m.RulesPath)
	return m
}

// Validate checks the values of the given metrics.
// If ok, return nil, otherwise returns an error.
func (m Metrics) Validate() error {
	return nil
}
