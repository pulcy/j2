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

var (
	unitSuffixes = []string{".service", ".timer"}
)

func IsUnitForJob(unitName string, jobName JobName) bool {
	prefix := fmt.Sprintf("%s-", jobName)
	if !strings.HasPrefix(unitName, prefix) {
		return false
	}
	found, _ := hasUnitSuffix(unitName)
	return found
}

func IsUnitForTaskGroup(unitName string, jobName JobName, taskGroupName TaskGroupName) bool {
	prefix := fmt.Sprintf("%s-%s-", jobName, taskGroupName)
	if !strings.HasPrefix(unitName, prefix) {
		return false
	}
	found, _ := hasUnitSuffix(unitName)
	return found
}

func IsUnitForTask(unitName string, jobName JobName, taskGroupName TaskGroupName, taskName TaskName) bool {
	prefix := fmt.Sprintf("%s-%s-%s", jobName, taskGroupName, taskName)
	if !strings.HasPrefix(unitName, prefix) {
		return false
	}
	if found, _ := hasUnitSuffix(unitName); !found {
		return false
	}
	remainder := unitName[len(prefix):]
	return strings.HasPrefix(remainder, "@") || strings.HasPrefix(remainder, ".")
}

func IsUnitForScalingGroup(unitName string, jobName JobName, scalingGroup uint) bool {
	prefix := fmt.Sprintf("%s-", jobName)
	if !strings.HasPrefix(unitName, prefix) {
		return false
	}
	found, suffix := hasUnitSuffix(unitName)
	if !found {
		return false
	}
	name := unitName[:len(unitName)-len(suffix)]
	scalingSuffix := fmt.Sprintf("@%d", scalingGroup)
	if strings.HasSuffix(name, scalingSuffix) {
		return true
	}
	if scalingGroup == 1 && !strings.Contains(name, "@") {
		return true
	}
	return false
}

func hasUnitSuffix(unitName string) (bool, string) {
	for _, suffix := range unitSuffixes {
		if strings.HasSuffix(unitName, suffix) {
			return true, suffix
		}
	}
	return false, ""
}
