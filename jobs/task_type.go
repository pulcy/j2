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

type TaskType string

func (tt TaskType) String() string {
	return string(tt)
}

func (tt TaskType) Validate() error {
	switch tt {
	case "", "oneshot", "proxy":
		return nil
	default:
		return maskAny(errgo.WithCausef(nil, ValidationError, "type has invalid value '%s'", tt))
	}
}

// IsDefault returns true if the given task type equals default ("")
func (tt TaskType) IsDefault() bool {
	return tt == ""
}

// IsOneshot returns true if the given task type equals "oneshot"
func (tt TaskType) IsOneshot() bool {
	return tt == "oneshot"
}

// IsProxy returns true if the given task type equals "proxy"
func (tt TaskType) IsProxy() bool {
	return tt == "proxy"
}
