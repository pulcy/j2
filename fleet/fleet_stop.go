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
	"bytes"
	"fmt"

	"github.com/coreos/fleet/job"
)

func (f *FleetTunnel) Stop(unitNames ...string) (string, error) {
	log.Debugf("stopping %v", unitNames)

	stdout := &bytes.Buffer{}

	units, err := f.findUnits(unitNames)
	if err != nil {
		return "", maskAny(err)
	}

	stopping := make([]string, 0)
	for _, u := range units {
		if !suToGlobal(u) {
			if job.JobState(u.CurrentState) == job.JobStateInactive {
				return "", maskAny(fmt.Errorf("Unable to stop unit %s in state %s", u.Name, job.JobStateInactive))
			} else if job.JobState(u.CurrentState) == job.JobStateLoaded {
				log.Debugf("Unit(%s) already %s, skipping.", u.Name, job.JobStateLoaded)
				continue
			}
		}

		log.Debugf("Setting target state of Unit(%s) to %s", u.Name, job.JobStateLoaded)
		f.cAPI.SetUnitTargetState(u.Name, string(job.JobStateLoaded))
		if suToGlobal(u) {
			stdout.WriteString(fmt.Sprintf("Triggered global unit %s stop\n", u.Name))
		} else {
			stopping = append(stopping, u.Name)
		}
	}

	if err := f.tryWaitForUnitStates(stopping, "stop", job.JobStateLoaded, f.BlockAttempts, stdout); err != nil {
		return "", maskAny(err)
	}
	stdout.WriteString(fmt.Sprintf("Successfully stopped units %v.\n", stopping))

	return stdout.String(), nil
}
