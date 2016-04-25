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
	RestartPolicyAll = RestartPolicy("all")
)

// RestartPolicy specifies how to restart tasks in a task group.
type RestartPolicy string

// String returns a restart policy as string
func (rp RestartPolicy) String() string {
	return string(rp)
}

// Validate checks if a restart policy follows a valid format
func (rp RestartPolicy) Validate() error {
	switch string(rp) {
	case "all", "":
		return nil
	default:
		return maskAny(errgo.WithCausef(nil, ValidationError, "invalid restart policy '%s'", string(rp)))
	}
}

func (rp RestartPolicy) IsAll() bool {
	return rp == "all"
}
