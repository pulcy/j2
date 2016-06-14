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
	"strings"

	"github.com/juju/errgo"

	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/pkg/sdunits"
)

func setupInstanceConstraints(t *jobs.Task, unit *sdunits.Unit, unitKind string, ctx generatorContext) error {
	unit.FleetOptions.IsGlobal = t.GroupGlobal()
	if ctx.InstanceCount > 1 {
		if t.GroupGlobal() {
			if t.GroupCount() > 1 {
				// Setup metadata constraint such that instances are only scheduled on some machines
				if int(t.GroupCount()) > len(ctx.FleetOptions.GlobalInstanceConstraints) {
					// Group count to high
					return maskAny(errgo.WithCausef(nil, ValidationError, "Group count (%d) higher than #global instance constraints (%d)", t.GroupCount(), len(ctx.FleetOptions.GlobalInstanceConstraints)))
				}
				constraint := ctx.FleetOptions.GlobalInstanceConstraints[ctx.ScalingGroup-1]
				unit.FleetOptions.MachineMetadata(constraint)
			}
		} else {
			unit.FleetOptions.Conflicts(unitNameExt(t, unitKind, "*") + ".service")
		}
	}
	return nil
}

// setupConstraints creates constraint keys for the `X-Fleet` section for the main unit
func setupConstraints(t *jobs.Task, unit *sdunits.Unit) error {
	constraints := t.MergedConstraints()

	metadata := []string{}
	for _, c := range constraints {
		if strings.HasPrefix(c.Attribute, jobs.MetaAttributePrefix) {
			// meta.<somekey>
			key := c.Attribute[len(jobs.MetaAttributePrefix):]
			metadata = append(metadata, fmt.Sprintf("%s=%s", key, c.Value))
		} else {
			switch c.Attribute {
			case jobs.AttributeNodeID:
				unit.FleetOptions.MachineID = c.Value
			default:
				return errgo.WithCausef(nil, ValidationError, "Unknown constraint attribute '%s'", c.Attribute)
			}
		}
	}
	unit.FleetOptions.MachineMetadata(metadata...)

	return nil
}
