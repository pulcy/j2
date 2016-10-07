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

	unit, err := createDefaultUnit(t,
		unitName(t, namePostfix, ctx.ScalingGroup),
		unitDescription(t, fmt.Sprintf("Volume %d", volIndex), ctx.ScalingGroup),
		"service", namePostfix, ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	cmds, err := engine.CreateVolumeCmds(t, vol, volIndex, volPrefix, volHostPath, unit.ExecOptions.Environment, ctx.ScalingGroup)
	if err != nil {
		return nil, maskAny(err)
	}
	setupUnitFromCmds(unit, cmds)

	return unit, nil
}

// createVolumeUnitNamePostfix creates the volume unit kind + volume-index
func createVolumeUnitNamePostfix(volIndex int) string {
	return fmt.Sprintf("%s%d", unitKindVolume, volIndex)
}
