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
	"github.com/pulcy/j2/pkg/sdunits"
)

// createProxyUnit
func createProxyUnit(t *jobs.Task, link jobs.Link, linkIndex int, engine engine.Engine, ctx generatorContext) (*sdunits.Unit, error) {
	namePostfix := fmt.Sprintf("%s%d", unitKindProxy, linkIndex)
	unit, err := createDefaultUnit(t,
		unitName(t, namePostfix, ctx.ScalingGroup),
		unitDescription(t, fmt.Sprintf("Proxy %d", linkIndex), ctx.ScalingGroup),
		"service", namePostfix, ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	unit.ExecOptions.StopWhenUnneeded()
	cmds, err := engine.CreateProxyCmds(t, link, linkIndex, unit.ExecOptions.Environment, ctx.ScalingGroup)
	if err != nil {
		return nil, maskAny(err)
	}
	setupUnitFromCmds(unit, cmds)

	return unit, nil
}
