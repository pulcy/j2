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

	"github.com/juju/errgo"
)

// LinkName is a name of a link consisting of:
// <job>.<task> or
// <job>.<taskgroup>.<task>
type LinkName string

// NewLinkName assembles a link name from its elements.
func NewLinkName(jn JobName, tgn TaskGroupName, tn TaskName) LinkName {
	return LinkName(fmt.Sprintf("%s.%s.%s", jn, tgn, tn))
}

// String returns a link name in format <job>.<taskgroup>.<task>
func (ln LinkName) String() string {
	jn, tgn, tn, err := ln.parse()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s.%s.%s", jn, tgn, tn)
}

// PrivateDomainName returns the DNS name (in the private namespace) for the given link name.
func (ln LinkName) PrivateDomainName() string {
	return fmt.Sprintf("%s.private", ln.String())
}

// Validate checks if a link name follows a valid format
func (ln LinkName) Validate() error {
	_, _, _, err := ln.parse()
	return maskAny(err)
}

func (ln LinkName) parse() (JobName, TaskGroupName, TaskName, error) {
	parts := strings.Split(string(ln), ".")
	var jobName JobName
	var taskGroupName TaskGroupName
	var taskName TaskName
	switch len(parts) {
	case 2:
		jobName = JobName(parts[0])
		taskGroupName = TaskGroupName(parts[1])
		taskName = TaskName(parts[1])
	case 3:
		jobName = JobName(parts[0])
		taskGroupName = TaskGroupName(parts[1])
		taskName = TaskName(parts[2])
	default:
		return "", "", "", maskAny(errgo.WithCausef(nil, InvalidNameError, "unrecognized link '%s'", string(ln)))
	}
	if err := jobName.Validate(); err != nil {
		return "", "", "", maskAny(err)
	}
	if err := taskGroupName.Validate(); err != nil {
		return "", "", "", maskAny(err)
	}
	if err := taskName.Validate(); err != nil {
		return "", "", "", maskAny(err)
	}
	return jobName, taskGroupName, taskName, nil
}

func (ln LinkName) normalize() LinkName {
	jn, tgn, tn, err := ln.parse()
	if err != nil {
		return ln
	}
	return NewLinkName(jn, tgn, tn)
}
