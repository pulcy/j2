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

package kubernetes

import (
	"github.com/pulcy/j2/render"
)

type unitData struct {
	name    string
	content string
}

func newUnitData(name, content string) render.UnitData {
	return &unitData{
		name:    name,
		content: content,
	}
}

func (u *unitData) String() string {
	return u.name
}

func (u *unitData) Name() string {
	return u.name
}
func (u *unitData) Content() string {
	return u.content
}
