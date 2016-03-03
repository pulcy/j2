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
// <job>.<task>[@<instance>] or
// <job>.<taskgroup>.<task>[@instance]
type LinkName string

// NewLinkName assembles a link name from its elements.
func NewLinkName(jn JobName, tgn TaskGroupName, tn TaskName, in InstanceName) LinkName {
	result := fmt.Sprintf("%s.%s.%s", jn, tgn, tn)
	if !in.IsEmpty() {
		result = fmt.Sprintf("%s@%s", result, in)
	}
	return LinkName(result)
}

// String returns a link name in format <job>.<taskgroup>.<task>
func (ln LinkName) String() string {
	jn, tgn, tn, in, err := ln.parse()
	if err != nil {
		return ""
	}
	return string(NewLinkName(jn, tgn, tn, in))
}

// PrivateDomainName returns the DNS name (in the private namespace) for the given link name.
func (ln LinkName) PrivateDomainName() string {
	return fmt.Sprintf("%s.private", strings.Replace(ln.String(), "@", ".", -1))
}

// etcdServiceName returns name of the service as it is used in ETCD.
func (ln LinkName) etcdServiceName() string {
	return strings.Replace(strings.Replace(ln.String(), ".", "-", -1), "@", "-", -1)
}

// Validate checks if a link name follows a valid format
func (ln LinkName) Validate() error {
	_, _, _, _, err := ln.parse()
	return maskAny(err)
}

func (ln LinkName) parse() (JobName, TaskGroupName, TaskName, InstanceName, error) {
	var instanceName InstanceName
	parts := strings.Split(string(ln), "@")
	switch len(parts) {
	case 1:
		// Empty instance name
	case 2:
		instanceName = InstanceName(parts[1])
	default:
		return "", "", "", "", maskAny(errgo.WithCausef(nil, InvalidNameError, "unrecognized link '%s'", string(ln)))
	}

	parts = strings.Split(parts[0], ".")
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
		return jobName, taskGroupName, taskName, instanceName,
			maskAny(errgo.WithCausef(nil, InvalidNameError, "unrecognized link '%s'", string(ln)))
	}
	if err := jobName.Validate(); err != nil {
		return jobName, taskGroupName, taskName, instanceName, maskAny(err)
	}
	if err := taskGroupName.Validate(); err != nil {
		return jobName, taskGroupName, taskName, instanceName, maskAny(err)
	}
	if err := taskName.Validate(); err != nil {
		return jobName, taskGroupName, taskName, instanceName, maskAny(err)
	}
	if err := instanceName.Validate(); err != nil {
		return jobName, taskGroupName, taskName, instanceName, maskAny(err)
	}
	return jobName, taskGroupName, taskName, instanceName, nil
}

func (ln LinkName) normalize() LinkName {
	jn, tgn, tn, in, err := ln.parse()
	if err != nil {
		return ln
	}
	return NewLinkName(jn, tgn, tn, in)
}
