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
	LinkTypeHTTP = LinkType("http")
	LinkTypeTCP  = LinkType("tcp")
)

// LinkType is a type of a link: http|tcp
type LinkType string

// String returns a link type as string
func (lt LinkType) String() string {
	return string(lt)
}

// Validate checks if a link name follows a valid format
func (lt LinkType) Validate() error {
	switch string(lt) {
	case "http", "tcp", "":
		return nil
	default:
		return maskAny(errgo.WithCausef(nil, ValidationError, "invalid link type '%s'", string(lt)))
	}
}

func (lt LinkType) IsHTTP() bool {
	switch string(lt) {
	case "http", "":
		return true
	default:
		return false
	}
}

func (lt LinkType) IsTCP() bool {
	return lt == "tcp"
}
