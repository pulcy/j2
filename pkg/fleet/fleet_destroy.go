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
	"time"

	"github.com/coreos/fleet/client"
	aerr "github.com/ewoutp/go-aggregate-error"
)

func (f *FleetTunnel) Destroy(events chan Event, unitNames ...string) error {
	log.Debugf("destroying %v", unitNames)

	var ae aerr.AggregateError

	for _, unit := range unitNames {
		events <- newEvent(unit, "destroying")
		err := f.cAPI.DestroyUnit(unit)
		if err != nil {
			// Ignore 'Unit does not exist' error
			if client.IsErrorUnitNotFound(err) {
				continue
			}
			ae.Add(maskAny(fmt.Errorf("Error destroying units: %v", err)))
			continue
		}

		if f.NoBlock {
			attempts := f.BlockAttempts
			retry := func() bool {
				if f.BlockAttempts < 1 {
					return true
				}
				attempts--
				if attempts == 0 {
					return false
				}
				return true
			}

			for retry() {
				u, err := f.cAPI.Unit(unit)
				if err != nil {
					ae.Add(maskAny(fmt.Errorf("Error destroying units: %v", err)))
					break
				}

				if u == nil {
					break
				}
				time.Sleep(defaultSleepTime)
			}
		}

		events <- newEvent(unit, "destroyed")
	}

	if !ae.IsEmpty() {
		return maskAny(&ae)
	}

	return nil
}
