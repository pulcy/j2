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

package render

import (
	"github.com/pulcy/j2/engine"
	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/pkg/sdunits"
)

// createMainUnit
func createMainUnit(t *jobs.Task, sidekickUnitNames []string, engine engine.Engine, ctx generatorContext) (*sdunits.Unit, error) {
	unit := &sdunits.Unit{
		Name:         unitName(t, unitKindMain, ctx.ScalingGroup),
		FullName:     unitName(t, unitKindMain, ctx.ScalingGroup) + ".service",
		Description:  unitDescription(t, "Main", ctx.ScalingGroup),
		Type:         "service",
		Scalable_:    true, //t.group.IsScalable(),
		ScalingGroup: ctx.ScalingGroup,
		ExecOptions:  sdunits.NewExecOptions(),
		FleetOptions: sdunits.NewFleetOptions(),
	}
	cmds, err := engine.CreateMainCmds(t, unit.ExecOptions.Environment, ctx.ScalingGroup)
	if err != nil {
		return nil, maskAny(err)
	}
	setupUnitFromCmds(unit, cmds)
	switch t.Type {
	case "oneshot":
		unit.ExecOptions.IsOneshot = true
		unit.ExecOptions.Restart = "on-failure"
	case "proxy":
		unit.ExecOptions.Restart = "always"
	default:
		unit.ExecOptions.Restart = "always"
	}

	if err := setupInstanceConstraints(t, unit, unitKindMain, ctx); err != nil {
		return nil, maskAny(err)
	}

	// Service dependencies
	unit.ExecOptions.Require(commonRequires...)
	unit.ExecOptions.Require(sidekickUnitNames...)
	unit.ExecOptions.After(commonAfter...)
	unit.ExecOptions.After(sidekickUnitNames...)
	for _, name := range t.VolumesFrom {
		other, err := t.Task(name)
		if err != nil {
			return nil, maskAny(err)
		}
		otherName := unitName(other, unitKindMain, ctx.ScalingGroup) + ".service"
		unit.ExecOptions.Require(otherName)
		unit.ExecOptions.After(otherName)
	}

	if err := addFrontEndRegistration(t, unit, ctx); err != nil {
		return nil, maskAny(err)
	}

	if err := setupConstraints(t, unit); err != nil {
		return nil, maskAny(err)
	}

	addFleetOptions(t, ctx.FleetOptions, unit)

	return unit, nil
}
