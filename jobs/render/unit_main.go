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
	unit, err := createDefaultUnit(t,
		unitName(t, unitKindMain, ctx.ScalingGroup),
		unitDescription(t, "Main", ctx.ScalingGroup),
		"service", unitKindMain, ctx)
	if err != nil {
		return nil, maskAny(err)
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

	// Additional service dependencies
	unit.ExecOptions.Require(sidekickUnitNames...)
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

	// Add frontend registration commands
	if err := addFrontEndRegistration(t, unit, ctx); err != nil {
		return nil, maskAny(err)
	}

	return unit, nil
}
