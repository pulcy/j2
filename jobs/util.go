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
	"strings"
)

const (
	shortArgLen = 4
)

// addArg creates an environment variable for the given argument and adds that environment variable to the given commandline.
// If the given argument is short or it contains a '$', it is added directly.
func addArg(arg string, cmd *[]string, env map[string]string) {
	if (len(arg) <= shortArgLen) || strings.Contains(arg, "$") {
		*cmd = append(*cmd, arg)
	} else {
		// Look for an existing environment variable with the same value
		key := ""
		for k, v := range env {
			if v == arg {
				key = k
				break
			}
		}
		if key == "" {
			// No environment variable with same value found, create a new one
			key = fmt.Sprintf("A%02d", len(env))
			env[key] = arg
		}
		// Add to commandline
		*cmd = append(*cmd, fmt.Sprintf("$%s", key))
	}
}
