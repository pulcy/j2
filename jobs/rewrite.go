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

type Rewrite struct {
	PathPrefix       string `json:"path-prefix,omitempty" mapstructure:"path-prefix,omitempty"`
	RemovePathPrefix string `json:"remove-path-prefix,omitempty" mapstructure:"remove-path-prefix,omitempty"`
	Domain           string `json:"domain,omitempty" mapstructure:"domain,omitempty"`
}

// HasPathPrefixOnly returns true if only `path` has a non-empty value.
func (r Rewrite) HasPathPrefixOnly() bool {
	return r.PathPrefix != "" && r.RemovePathPrefix == "" && r.Domain == ""
}

func (r Rewrite) replaceVariables(ctx *variableContext) Rewrite {
	r.PathPrefix = ctx.replaceString(r.PathPrefix)
	r.RemovePathPrefix = ctx.replaceString(r.RemovePathPrefix)
	r.Domain = ctx.replaceString(r.Domain)
	return r
}

// Merge merges non-empty data from other into the given Rewrite.
// Data from the given Rewrite prevails over the other Rewrite.
func (r Rewrite) Merge(other Rewrite) Rewrite {
	if r.PathPrefix == "" {
		r.PathPrefix = other.PathPrefix
	}
	if r.RemovePathPrefix == "" {
		r.RemovePathPrefix = other.RemovePathPrefix
	}
	if r.Domain == "" {
		r.Domain = other.Domain
	}
	return r
}
