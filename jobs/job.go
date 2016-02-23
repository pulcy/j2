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
	"encoding/json"
	"regexp"
	"sort"

	"github.com/juju/errgo"
)

var (
	jobNamePattern = regexp.MustCompile(`^([a-z0-9_]{3,30})$`)
)

type JobName string

func (jn JobName) String() string {
	return string(jn)
}

func (jn JobName) Validate() error {
	if !jobNamePattern.MatchString(string(jn)) {
		return maskAny(errgo.WithCausef(nil, InvalidNameError, "job name must match '%s', got '%s'", jobNamePattern, jn))
	}
	return nil
}

type Job struct {
	ID          string        `json:"id,omitempty"`
	Name        JobName       `json:"name"`
	Groups      TaskGroupList `json:"groups"`
	Constraints Constraints   `json:"constraints,omitempty"`
}

// Link objects just after parsing
func (j *Job) link() {
	for _, tg := range j.Groups {
		tg.job = j
		tg.link()
	}
	sort.Sort(j.Groups)
}

// Check for errors
func (j *Job) Validate() error {
	if err := j.Name.Validate(); err != nil {
		return maskAny(err)
	}
	if len(j.Groups) == 0 {
		return maskAny(errgo.WithCausef(nil, ValidationError, "job has no groups"))
	}
	for i, tg := range j.Groups {
		err := tg.Validate()
		if err != nil {
			return maskAny(err)
		}
		for k := i + 1; k < len(j.Groups); k++ {
			if j.Groups[k].Name == tg.Name {
				return maskAny(errgo.WithCausef(nil, ValidationError, "job has duplicate taskgroup %s", tg.Name))
			}
		}
	}
	if err := j.Constraints.Validate(); err != nil {
		return maskAny(err)
	}
	return nil
}

func (j *Job) Generate(config GeneratorConfig) *Generator {
	return newGenerator(j, config)
}

func (j *Job) MaxCount() uint {
	count := uint(0)
	for _, tg := range j.Groups {
		if tg.Count > count {
			count = tg.Count
		}
	}
	return count
}

// Json returns formatted json representation of this job.
func (j *Job) Json() ([]byte, error) {
	json, err := json.MarshalIndent(j, "", "\t")
	if err != nil {
		return []byte(""), maskAny(err)
	}
	return json, nil
}
