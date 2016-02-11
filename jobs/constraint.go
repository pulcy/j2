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

const (
	metaAttributePrefix = "meta."
	attributeNodeID     = "node.id"
)

// Constraint contains a specification of a scheduling constraint.
type Constraint struct {
	Attribute string `json:"attribute,omitempty" mapstructure:"attribute,omitempty"`
	Value     string `json:"value,omitempty" mapstructure:"value,omitempty"`
}

// Validate checks the values of the given constraint.
// If ok, return nil, otherwise returns an error.
func (c Constraint) Validate() error {
	if c.Attribute == "" {
		return errgo.WithCausef(nil, ValidationError, "attribute cannot be empty")
	}
	return nil
}

// Constraints is a list of Constraint's
type Constraints []Constraint

// Validate checks the values of all constraints in the given list.
// If ok, return nil, otherwise returns an error.
func (list Constraints) Validate() error {
	attributes := make(map[string]struct{})
	for _, c := range list {
		if err := c.Validate(); err != nil {
			return maskAny(err)
		}
		if _, ok := attributes[c.Attribute]; ok {
			return errgo.WithCausef(nil, ValidationError, "duplicate constraint for attribute '%s'", c.Attribute)
		}
		attributes[c.Attribute] = struct{}{}
	}
	return nil
}

// Contains returns true if the given list contains a constrains with the given attribute.
// Otherwise it returns false.
func (list Constraints) Contains(attribute string) bool {
	for _, c := range list {
		if c.Attribute == attribute {
			return true
		}
	}
	return false
}

// Merge creates a new list of constraints with all constraints in `list` combined with all constraints
// of `additional`. If attributes exists in both lists, the attribute in `additional` wins.
func (list Constraints) Merge(additional Constraints) Constraints {
	result := append(Constraints{}, additional...)
	for _, c := range list {
		if !additional.Contains(c.Attribute) {
			result = append(result, c)
		}
	}
	return result
}
