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
	"sync"

	"github.com/coreos/fleet/api"
	"github.com/coreos/fleet/job"
	"github.com/coreos/fleet/schema"
	"github.com/coreos/fleet/unit"
	aerr "github.com/ewoutp/go-aggregate-error"
)

type UnitData interface {
	Name() string
	Content() string
}

type UnitDataList interface {
	Len() int
	Get(index int) UnitData
}

func (f *FleetTunnel) Start(events chan Event, units UnitDataList) error {
	log.Debugf("starting %v", units)

	if err := f.lazyCreateUnits(units, events); err != nil {
		return maskAny(fmt.Errorf("Error creating units: %v", err))
	}

	triggered, err := f.lazyStartUnits(units)
	if err != nil {
		return maskAny(fmt.Errorf("Error starting units: %v", err))
	}

	var starting []string
	for _, u := range triggered {
		if suToGlobal(*u) {
			events <- newEvent(u.Name, "triggered global unit start")
		} else {
			starting = append(starting, u.Name)
		}
	}

	if err := f.tryWaitForUnitStates(starting, "start", job.JobStateLaunched, f.BlockAttempts, events); err != nil {
		return maskAny(err)
	}
	return nil
}

// lazyCreateUnits iterates over a set of unit names and, for each, attempts to
// ensure that a unit by that name exists in the Registry, by checking a number
// of conditions and acting on the first one that succeeds, in order of:
//  1. a unit by that name already existing in the Registry
//  2. a unit file by that name existing on disk
//  3. a corresponding unit template (if applicable) existing in the Registry
//  4. a corresponding unit template (if applicable) existing on disk
// Any error encountered during these steps is returned immediately (i.e.
// subsequent Jobs are not acted on). An error is also returned if none of the
// above conditions match a given Job.
func (f *FleetTunnel) lazyCreateUnits(units UnitDataList, events chan Event) error {
	errchan := make(chan error)
	blockAttempts := f.BlockAttempts
	var wg sync.WaitGroup
	for i := 0; i < units.Len(); i++ {
		u := units.Get(i)
		name := u.Name()
		create, err := f.checkUnitCreation(name)
		if err != nil {
			return err
		} else if !create {
			continue
		}

		// Assume that the name references a local unit file on
		// disk or if it is an instance unit and if so get its
		// corresponding unit
		uf, err := unit.NewUnitFile(u.Content())
		if err != nil {
			return err
		}

		events <- newEvent(name, "creating unit")
		_, err = f.createUnit(name, uf)
		if err != nil {
			return err
		}

		wg.Add(1)
		go f.checkUnitState(name, job.JobStateInactive, blockAttempts, events, &wg, errchan)
	}

	go func() {
		wg.Wait()
		close(errchan)
	}()

	var ae aerr.AggregateError
	for msg := range errchan {
		ae.Add(maskAny(fmt.Errorf("Error waiting on unit creation: %v\n", msg)))
	}

	if !ae.IsEmpty() {
		return maskAny(&ae)
	}

	return nil
}

// checkUnitCreation checks if the unit with the given name should be created.
func (f *FleetTunnel) checkUnitCreation(unitName string) (bool, error) {
	// First, check if there already exists a Unit by the given name in the Registry
	exists, err := f.unitExists(unitName)
	if err != nil {
		return false, maskAny(fmt.Errorf("error retrieving Unit(%s) from Registry: %v", unitName, err))
	}
	return !exists, nil
}

func (f *FleetTunnel) lazyStartUnits(units UnitDataList) ([]*schema.Unit, error) {
	unitNames := make([]string, 0, units.Len())
	for i := 0; i < units.Len(); i++ {
		unitNames = append(unitNames, units.Get(i).Name())
	}
	return f.setTargetStateOfUnits(unitNames, job.JobStateLaunched)
}

func (f *FleetTunnel) createUnit(name string, uf *unit.UnitFile) (*schema.Unit, error) {
	if uf == nil {
		return nil, maskAny(fmt.Errorf("nil unit provided"))
	}
	u := schema.Unit{
		Name:    name,
		Options: schema.MapUnitFileToSchemaUnitOptions(uf),
	}
	// TODO(jonboulle): this dependency on the API package is awkward, and
	// redundant with the check in api.unitsResource.set, but it is a
	// workaround to implementing the same check in the RegistryClient. It
	// will disappear once RegistryClient is deprecated.
	if err := api.ValidateName(name); err != nil {
		return nil, maskAny(err)
	}
	if err := api.ValidateOptions(u.Options); err != nil {
		return nil, maskAny(err)
	}
	j := &job.Job{Unit: *uf}
	if err := j.ValidateRequirements(); err != nil {
		log.Warningf("Unit %s: %v", name, err)
	}
	err := f.createUnitWithRetry(&u)
	if err != nil {
		return nil, maskAny(fmt.Errorf("failed creating unit %s: %v", name, err))
	}

	log.Debugf("Created Unit(%s) in Registry", name)
	return &u, nil
}
