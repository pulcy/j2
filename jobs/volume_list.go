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
)

// VolumeList is a list of volumes, ordered such that volumes that need a volume unit come first
type VolumeList []Volume

// Len is the number of elements in the collection.
func (l VolumeList) Len() int {
	return len(l)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (l VolumeList) Less(i, j int) bool {
	vi := l[i]
	vj := l[j]
	if !vi.IsLocal() && vj.IsLocal() {
		return true
	} else if vi.IsLocal() && !vj.IsLocal() {
		return false
	}
	return strings.Compare(vi.Path, vj.Path) < 0
}

// Swap swaps the elements with indexes i and j.
func (l VolumeList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}
