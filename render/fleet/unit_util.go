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

	"github.com/pulcy/j2/engine"
	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/pkg/cmdline"
	"github.com/pulcy/j2/pkg/sdunits"
)

func createBaseUnit(t *jobs.Task, name, description, unitType string, ctx generatorContext) *sdunits.Unit {
	unit := &sdunits.Unit{
		Name:         name,
		FullName:     fmt.Sprintf("%s.%s", name, unitType),
		Description:  description,
		Type:         unitType,
		ScalingGroup: ctx.ScalingGroup,
		ExecOptions:  sdunits.NewExecOptions(),
		FleetOptions: sdunits.NewFleetOptions(),
	}
	addFleetOptions(t, ctx.FleetOptions, unit)

	return unit
}

func createDefaultUnit(t *jobs.Task, name, description, unitType, unitKind string, ctx generatorContext) (*sdunits.Unit, error) {
	unit := createBaseUnit(t, name, description, unitType, ctx)
	unit.ExecOptions.Restart = "always"

	if err := setupInstanceConstraints(t, unit, unitKind, ctx); err != nil {
		return nil, maskAny(err)
	}

	// Service dependencies
	unit.ExecOptions.Require(commonRequires...)
	unit.ExecOptions.After(commonAfter...)
	if t.Network.IsWeave() {
		unit.ExecOptions.Require(weaveRequires...)
		unit.ExecOptions.After(weaveAfter...)
	}

	if err := setupConstraints(t, unit); err != nil {
		return nil, maskAny(err)
	}

	return unit, nil
}

func setupUnitFromCmds(unit *sdunits.Unit, cmds engine.Cmds) {
	formatCmd := func(cmd cmdline.Cmdline) string {
		result := cmd.String()
		if cmd.AllowFailure {
			result = "-" + result
		}
		return result
	}

	if len(cmds.Start) > 0 {
		for i := 0; i < len(cmds.Start)-1; i++ {
			unit.ExecOptions.ExecStartPre = append(unit.ExecOptions.ExecStartPre, formatCmd(cmds.Start[i]))
		}
		unit.ExecOptions.ExecStart = formatCmd(cmds.Start[len(cmds.Start)-1])
	}
	if len(cmds.Stop) > 0 {
		unit.ExecOptions.ExecStop = append(unit.ExecOptions.ExecStop, formatCmd(cmds.Stop[0]))
		for i := 1; i < len(cmds.Stop); i++ {
			unit.ExecOptions.ExecStopPost = append(unit.ExecOptions.ExecStopPost, formatCmd(cmds.Stop[i]))
		}
	}
}
