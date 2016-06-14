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
	"github.com/cenkalti/backoff"
	"github.com/coreos/fleet/schema"
)

func (f *FleetTunnel) Status() (StatusMap, error) {
	log.Debugf("list unit status")

	var states []*schema.UnitState
	op := func() error {
		var err error
		states, err = f.cAPI.UnitStates()
		if err != nil {
			return maskAny(err)
		}
		return nil
	}
	if err := backoff.Retry(op, backoff.NewExponentialBackOff()); err != nil {
		return StatusMap{}, maskAny(err)
	}

	return newStatusMapFromUnits(states), nil
}
