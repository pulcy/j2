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
	"github.com/juju/errgo"
)

type Link struct {
	Target LinkName `json:"target"`
	Type   LinkType `json:"type,omitempty" mapstructure:"type,omitempty"`
	Ports  []int    `json:"ports,omitempty" mapstructure:"ports,omitempty"`
}

func (l Link) Validate() error {
	if err := l.Target.Validate(); err != nil {
		return maskAny(err)
	}
	if err := l.Type.Validate(); err != nil {
		return maskAny(err)
	}
	if len(l.Ports) == 0 && l.Type.IsTCP() {
		return maskAny(errgo.WithCausef(nil, ValidationError, "specify at least one port in a tcp link"))
	}
	if len(l.Ports) != 0 && !l.Type.IsTCP() {
		return maskAny(errgo.WithCausef(nil, ValidationError, "ports are not allowed in non-tcp links"))
	}
	return nil
}
