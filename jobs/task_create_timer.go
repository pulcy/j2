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
	"strconv"

	"arvika.pulcy.com/pulcy/deployit/units"
)

// createTimerUnit
func (t *Task) createTimerUnit(ctx generatorContext) (*units.Unit, error) {
	if t.Timer == "" {
		return nil, nil
	}
	unit := &units.Unit{
		Name:         t.unitName(unitKindTimer, strconv.Itoa(int(ctx.ScalingGroup))),
		FullName:     t.unitName(unitKindTimer, strconv.Itoa(int(ctx.ScalingGroup))) + ".timer",
		Description:  t.unitDescription("Timer", ctx.ScalingGroup),
		Type:         "timer",
		Scalable:     t.group.IsScalable(),
		ScalingGroup: ctx.ScalingGroup,
		ExecOptions:  units.NewExecOptions(),
		FleetOptions: units.NewFleetOptions(),
	}
	unit.ExecOptions.OnCalendar = t.Timer
	unit.ExecOptions.Unit = t.unitName(unitKindMain, strconv.Itoa(int(ctx.ScalingGroup))) + ".service"

	return unit, nil
}
