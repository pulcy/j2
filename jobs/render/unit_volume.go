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
	"fmt"
	"path"

	"github.com/pulcy/j2/engine"
	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/pkg/sdunits"
)

// createVolumeUnit
func createVolumeUnit(t *jobs.Task, vol jobs.Volume, volIndex int, engine engine.Engine, ctx generatorContext) (*sdunits.Unit, error) {
	namePostfix := createVolumeUnitNamePostfix(volIndex)
	volPrefix := path.Join(fmt.Sprintf("%s/%v", t.FullName(), ctx.ScalingGroup), vol.Path)
	volHostPath := fmt.Sprintf("/media/%s", volPrefix)

	unit := &sdunits.Unit{
		Name:         unitName(t, namePostfix, ctx.ScalingGroup),
		FullName:     unitName(t, namePostfix, ctx.ScalingGroup) + ".service",
		Description:  unitDescription(t, fmt.Sprintf("Volume %d", volIndex), ctx.ScalingGroup),
		Type:         "service",
		Scalable_:    true, //t.group.IsScalable(),
		ScalingGroup: ctx.ScalingGroup,
		ExecOptions:  sdunits.NewExecOptions(),
		FleetOptions: sdunits.NewFleetOptions(),
	}
	cmds, err := engine.CreateVolumeCmds(t, vol, volIndex, volPrefix, volHostPath, unit.ExecOptions.Environment, ctx.ScalingGroup)
	if err != nil {
		return nil, maskAny(err)
	}
	setupUnitFromCmds(unit, cmds)
	unit.ExecOptions.Restart = "always"

	if err := setupInstanceConstraints(t, unit, namePostfix, ctx); err != nil {
		return nil, maskAny(err)
	}

	// Service dependencies
	unit.ExecOptions.Require(commonRequires...)
	unit.ExecOptions.After(commonAfter...)

	if err := setupConstraints(t, unit); err != nil {
		return nil, maskAny(err)
	}

	addFleetOptions(t, ctx.FleetOptions, unit)

	return unit, nil
}

// createVolumeUnitNamePostfix creates the volume unit kind + volume-index
func createVolumeUnitNamePostfix(volIndex int) string {
	return fmt.Sprintf("%s%d", unitKindVolume, volIndex)
}
