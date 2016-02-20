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
	"fmt"
)

type LogDriver string

const (
	LogDriverNone = LogDriver("none")
)

func (lg LogDriver) String() string {
	return string(lg)
}

func (lg LogDriver) Validate() error {
	switch lg {
	case "", "none":
		return nil
	default:
		return maskAny(fmt.Errorf("unknown log-driver '%s'", lg))
	}
}

// CreateDockerLogArgs creates a series of command line arguments for the given
// log driver, based on the given cluster.
func (lg LogDriver) CreateDockerLogArgs(clusterDockerLoggingArgs []string) []string {
	switch lg {
	case "": // Default from cluster
		return clusterDockerLoggingArgs
	case "none":
		return nil
	default:
		// We already validate it, so this should not happen.
		// Just return an empty list
		return nil
	}
}
