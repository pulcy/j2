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
	"bytes"
)

type TaskList []*Task

func (l TaskList) Len() int {
	return len(l)
}

func (l TaskList) Less(i, j int) bool {
	return bytes.Compare([]byte(l[i].Name.String()), []byte(l[j].Name.String())) < 0
}

func (l TaskList) Swap(i, j int) {
	tmp := l[i]
	l[i] = l[j]
	l[j] = tmp
}

// IndexByName returns the index of the task with given name in the given list.
// Returns -1 if no such task is found.
func (l TaskList) IndexByName(taskName TaskName) int {
	for i, t := range l {
		if t.Name == taskName {
			return i
		}
	}
	return -1
}
