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

package fleet

import (
	_ "fmt"
	"strings"

	"github.com/coreos/fleet/schema"
)

type StatusMap struct {
	state map[string]string
}

func (s StatusMap) Get(unitName string) (string, bool) {
	if status, ok := s.state[unitName]; ok {
		return status, true
	}
	return "", false
}

func newStatusMapFromUnits(unitStates []*schema.UnitState) StatusMap {
	//fmt.Printf("Fleet Status:\n%s\n", listUnitsOutput)
	s := StatusMap{
		state: make(map[string]string),
	}
	for _, unit := range unitStates {
		s.state[unit.Name] = unit.SystemdActiveState
	}
	return s
}

func newStatusMap(listUnitsOutput string) StatusMap {
	//fmt.Printf("Fleet Status:\n%s\n", listUnitsOutput)
	s := StatusMap{
		state: make(map[string]string),
	}
	for _, line := range strings.Split(listUnitsOutput, "\n") {
		parts := splitStatusLine(line)
		//fmt.Printf("parts=%#v\n", parts)
		if len(parts) != 2 {
			continue
		}
		s.state[parts[0]] = parts[1]
	}
	return s
}

func splitStatusLine(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.Replace(line, "\t", " ", -1)
	parts := strings.Split(line, " ")

	result := []string{}
	for _, x := range parts {
		x = strings.TrimSpace(x)
		if x != "" {
			result = append(result, x)
		}
	}
	return result
}
