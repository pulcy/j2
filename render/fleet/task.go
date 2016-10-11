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
	"strconv"
	"strings"

	"github.com/pulcy/j2/engine"
	"github.com/pulcy/j2/engine/docker"
	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/pkg/sdunits"
)

const (
	unitKindMain   = "-mn"
	unitKindVolume = "-vl"
	unitKindProxy  = "-pr"
	unitKindTimer  = "-ti"
)

var (
	commonAfter = []string{
		"docker.service",
	}
	commonRequires = []string{
		"docker.service",
	}
	weaveAfter = []string{
		"weave.service",
	}
	weaveRequires = []string{
		"weave.service",
	}
)

// createTaskUnits creates all units needed to run this task.
func createTaskUnits(t *jobs.Task, ctx generatorContext) ([]sdunits.UnitChain, error) {
	mainChain := sdunits.UnitChain{}

	engine := newEngine(t, ctx)
	sidekickUnitNames := []string{}
	for _, l := range t.Links {
		if !l.Type.IsTCP() {
			continue
		}
		linkIndex := len(sidekickUnitNames)
		unit, err := createProxyUnit(t, l, linkIndex, engine, ctx)
		if err != nil {
			return nil, maskAny(err)
		}
		sidekickUnitNames = append(sidekickUnitNames, unit.FullName)
		mainChain = append(mainChain, unit)
	}

	for i, v := range t.Volumes {
		if v.IsLocal() {
			continue
		}
		unit, err := createVolumeUnit(t, v, i, engine, ctx)
		if err != nil {
			return nil, maskAny(err)
		}
		sidekickUnitNames = append(sidekickUnitNames, unit.FullName)
		mainChain = append(mainChain, unit)
	}

	main, err := createMainUnit(t, sidekickUnitNames, engine, ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	mainChain = append(mainChain, main)

	timer, err := createTimerUnit(t, ctx)
	if err != nil {
		return nil, maskAny(err)
	}

	chains := []sdunits.UnitChain{mainChain}
	if timer != nil {
		timerChain := sdunits.UnitChain{timer}
		chains = append(chains, timerChain)
	}

	return chains, nil
}

// newEngine creates a new Engine for the given task.
func newEngine(t *jobs.Task, ctx generatorContext) engine.Engine {
	return docker.NewDockerEngine(ctx.DockerOptions)
}

// unitName returns the name of the systemd unit for this task.
func unitName(t *jobs.Task, kind string, scalingGroup uint) string {
	return unitNameExt(t, kind, strconv.Itoa(int(scalingGroup)))
}

// unitNameExt returns the name of the systemd unit for this task.
func unitNameExt(t *jobs.Task, kind string, scalingGroup string) string {
	base := strings.Replace(t.FullName(), "/", "-", -1) + kind
	if t.GroupGlobal() && t.GroupCount() == 1 {
		return base
	}
	return fmt.Sprintf("%s@%v", base, scalingGroup)
}

// unitDescription creates the description of a unit
func unitDescription(t *jobs.Task, prefix string, scalingGroup uint) string {
	descriptionPostfix := fmt.Sprintf("[slice %d]", scalingGroup)
	if t.GroupGlobal() {
		descriptionPostfix = "[global]"
	}
	return fmt.Sprintf("%s unit for %s %s", prefix, t.FullName(), descriptionPostfix)
}
