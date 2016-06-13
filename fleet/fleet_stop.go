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
	"fmt"

	"github.com/coreos/fleet/job"
)

type StopStats struct {
	StoppedUnits       int
	StoppedGlobalUnits int
}

func (f *FleetTunnel) Stop(events chan Event, unitNames ...string) (StopStats, error) {
	log.Debugf("stopping %v", unitNames)

	units, err := f.findUnits(unitNames)
	if err != nil {
		return StopStats{}, maskAny(err)
	}

	stopping := make([]string, 0)
	stats := StopStats{}
	for _, u := range units {
		if !suToGlobal(u) {
			if job.JobState(u.CurrentState) == job.JobStateInactive {
				return StopStats{}, maskAny(fmt.Errorf("Unable to stop unit %s in state %s", u.Name, job.JobStateInactive))
			} else if job.JobState(u.CurrentState) == job.JobStateLoaded {
				log.Debugf("Unit(%s) already %s, skipping.", u.Name, job.JobStateLoaded)
				continue
			}
		}

		log.Debugf("Setting target state of Unit(%s) to %s", u.Name, job.JobStateLoaded)
		f.cAPI.SetUnitTargetState(u.Name, string(job.JobStateLoaded))
		if suToGlobal(u) {
			stats.StoppedGlobalUnits++
			events <- newEvent(u.Name, "triggered global unit stop")
		} else {
			stats.StoppedUnits++
			events <- newEvent(u.Name, "triggered unit stop")
			stopping = append(stopping, u.Name)
		}
	}

	if err := f.tryWaitForUnitStates(stopping, "stop", job.JobStateLoaded, f.BlockAttempts, events); err != nil {
		return StopStats{}, maskAny(err)
	}

	return stats, nil
}
