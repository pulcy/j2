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

import "strings"

// Dependency configures a link to a task in another job.
type Dependency struct {
	Name             LinkName          `json:"-", mapstructure:"-"`
	Network          NetworkType       `json:"network,omitempty" mapstructure:"network,omitempty"`
	PrivateFrontEnds []PrivateFrontEnd `json:"private-frontends,omitempty"`
}

func (t Dependency) Validate() error {
	if err := t.Name.Validate(); err != nil {
		return maskAny(err)
	}
	if err := t.Network.Validate(); err != nil {
		return maskAny(err)
	}
	return nil
}

func (t Dependency) replaceVariables(ctx *variableContext) Dependency {
	t.Name = LinkName(ctx.replaceString(string(t.Name)))
	t.Network = NetworkType(ctx.replaceString(string(t.Network)))
	for i, x := range t.PrivateFrontEnds {
		t.PrivateFrontEnds[i] = x.replaceVariables(ctx)
	}
	return t
}

// PrivateFrontEndPort returns the port number of the first private frontend with a non-0 port number.
func (t Dependency) PrivateFrontEndPort(defaultPort int) int {
	for _, f := range t.PrivateFrontEnds {
		if f.Port != 0 {
			return f.Port
		}
	}
	return defaultPort
}

type DependencyList []Dependency

func (l DependencyList) Validate() error {
	for _, d := range l {
		if err := d.Validate(); err != nil {
			return maskAny(err)
		}
	}
	return nil
}

func (l DependencyList) replaceVariables(ctx *variableContext) {
	for i, d := range l {
		l[i] = d.replaceVariables(ctx)
	}
}

// Len is the number of elements in the collection.
func (list DependencyList) Len() int {
	return len(list)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (list DependencyList) Less(i, j int) bool {
	return strings.Compare(string(list[i].Name), string(list[j].Name)) < 0
}

// Swap swaps the elements with indexes i and j.
func (list DependencyList) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}
