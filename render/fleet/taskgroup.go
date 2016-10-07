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
	"github.com/juju/errgo"

	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/pkg/sdunits"
)

type taskUnitChain struct {
	Task      *jobs.Task
	MainChain sdunits.UnitChain
}

type taskUnitChainList []taskUnitChain

func (l taskUnitChainList) find(taskName jobs.TaskName) (taskUnitChain, error) {
	for _, x := range l {
		if x.Task.Name == taskName {
			return x, nil
		}
	}
	return taskUnitChain{}, maskAny(errgo.WithCausef(nil, TaskNotFoundError, taskName.String()))
}

// createUnits creates all units needed to run this taskgroup.
func createTaskGroupUnits(tg *jobs.TaskGroup, ctx generatorContext) ([]sdunits.UnitChain, error) {
	if ctx.ScalingGroup > tg.Count {
		return nil, nil
	}

	// Create all units for my tasks
	allChains := []sdunits.UnitChain{}
	taskUnitChains := taskUnitChainList{}
	allUnits := []*sdunits.Unit{}
	for _, t := range tg.Tasks {
		tuc, err := createTaskUnits(t, ctx)
		if err != nil {
			return nil, maskAny(err)
		}
		allChains = append(allChains, tuc...)
		if len(tuc) > 0 {
			taskUnitChains = append(taskUnitChains, taskUnitChain{
				Task:      t,
				MainChain: tuc[0],
			})
		}
		// Link chains to enfore the actual chain
		for _, chain := range tuc {
			chain.Link()
		}
		// Collect all units in the chain
		for _, chain := range tuc {
			allUnits = append(allUnits, chain...)
		}
	}

	// In case of restart="all", bind chains such that they restart together
	if tg.RestartPolicy.IsAll() {
		for i, x := range taskUnitChains {
			for j, y := range taskUnitChains {
				if i == j {
					continue
				}
				x.MainChain.BindRestartTo(y.MainChain)
			}
		}
	}

	// Create "After" links
	for _, x := range taskUnitChains {
		for _, afterName := range x.Task.After {
			other, err := taskUnitChains.find(afterName)
			if err != nil {
				return nil, maskAny(err)
			}
			x.MainChain.After(other.MainChain)
		}
	}

	// Force units to be on the same machine
	if !tg.Global {
		sdunits.GroupUnitsOnMachine(allUnits...)
	}

	return allChains, nil
}
