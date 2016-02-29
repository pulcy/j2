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
	"strings"

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

// Equals returns true if the given constraints are exactly the same.
func (c Constraint) Equals(other Constraint) bool {
	return c.Attribute == other.Attribute &&
		c.Value == other.Value
}

// Conflicts returns true if the given constraints have the same attribute, but a different value.
func (c Constraint) Conflicts(other Constraint) bool {
	return c.Attribute == other.Attribute &&
		c.Value != other.Value
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
	_, found := list.get(attribute)
	return found
}

// get returns the constraint with given attribute from the given list.
func (list Constraints) get(attribute string) (Constraint, bool) {
	for _, c := range list {
		if c.Attribute == attribute {
			return c, true
		}
	}
	return Constraint{}, false
}

// Add creates a new list of constraints with all constraints in `list` combined with all constraints
// of `additional`. If attributes exists in both lists, an error is raised.
func (list Constraints) Add(additional Constraints) (Constraints, error) {
	result := append(Constraints{}, additional...)
	for _, c := range list {
		if other, found := additional.get(c.Attribute); found {
			if c.Conflicts(other) {
				return nil, maskAny(errgo.WithCausef(nil, ValidationError, "constraints '%s' has conflicting values '%s' and '%s'", c.Attribute, c.Value, other.Value))
			}
		}
		result = append(result, c)
	}
	return result, nil
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

// Len is the number of elements in the collection.
func (list Constraints) Len() int {
	return len(list)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (list Constraints) Less(i, j int) bool {
	return strings.Compare(list[i].Attribute, list[j].Attribute) < 0
}

// Swap swaps the elements with indexes i and j.
func (list Constraints) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}
