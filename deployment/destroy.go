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

package deployment

import (
	"fmt"
	"strings"
	"time"

	"github.com/pulcy/j2/fleet"
)

// Destroy removes all unit files that belong to the configured job from the configured cluster.
func (d *Deployment) Destroy(deps DeploymentDependencies) error {
	f := d.newFleetTunnel()
	list, err := f.List()
	if err != nil {
		return maskAny(err)
	}

	predicate := d.createUnitNamePredicate()
	unitNames := selectUnitNames(list, predicate)
	if len(unitNames) == 0 {
		fmt.Printf("No units on the cluster match the given arguments\n")
		return nil
	}

	if err := d.confirmDestroy(deps, unitNames, false); err != nil {
		return maskAny(err)
	}
	if err := d.destroyUnits(f, unitNames); err != nil {
		return maskAny(err)
	}

	return nil
}

func (d *Deployment) confirmDestroy(deps DeploymentDependencies, units []string, obsolete bool) error {
	if !d.force {
		obsoleteMsg := ""
		if obsolete {
			obsoleteMsg = " obsolete units"
		}
		if err := deps.Confirm(fmt.Sprintf("You are about to destroy%s:\n- %s\n\nAre you sure you want to destroy %d units on stack '%s'?\nEnter yes:", obsoleteMsg, strings.Join(units, "\n- "), len(units), d.cluster.Stack)); err != nil {
			return maskAny(err)
		}
		fmt.Println()
	}

	return nil
}

func (d *Deployment) destroyUnits(f fleet.FleetTunnel, units []string) error {
	if len(units) == 0 {
		return maskAny(fmt.Errorf("No units on cluster: %s", d.cluster.Stack))
	}

	var out string
	out, err := f.Stop(units...)
	if err != nil {
		fmt.Printf("Warning: stop failed.\n%s\n", err.Error())
	}

	fmt.Println(out)

	fmt.Printf("Waiting for %s...\n", d.StopDelay)
	time.Sleep(d.StopDelay)

	out, err = f.Destroy(units...)
	if err != nil {
		return maskAny(err)
	}

	fmt.Println(out)

	return nil
}
