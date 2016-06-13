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
	"io/ioutil"
	"os"
	"path"
	"sync"

	"github.com/coreos/fleet/api"
	"github.com/coreos/fleet/job"
	"github.com/coreos/fleet/schema"
	"github.com/coreos/fleet/unit"
	aerr "github.com/ewoutp/go-aggregate-error"
)

func (f *FleetTunnel) Start(events chan Event, units ...string) error {
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
func (f *FleetTunnel) lazyCreateUnits(args []string, events chan Event) error {
	errchan := make(chan error)
	blockAttempts := f.BlockAttempts
	var wg sync.WaitGroup
	for _, arg := range args {
		name := path.Base(arg)

		create, err := f.checkUnitCreation(arg)
		if err != nil {
			return err
		} else if !create {
			continue
		}

		// Assume that the name references a local unit file on
		// disk or if it is an instance unit and if so get its
		// corresponding unit
		uf, err := getUnitFile(arg)
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

// checkUnitCreation checks if the unit should be created.
// It takes a unit file path as a parameter.
// It returns 0 on success and if the unit should be created, 1 if the
// unit should not be created; and any error encountered.
func (f *FleetTunnel) checkUnitCreation(arg string) (bool, error) {
	name := path.Base(arg)

	// First, check if there already exists a Unit by the given name in the Registry
	unit, err := f.cAPI.Unit(name)
	if err != nil {
		return false, maskAny(fmt.Errorf("error retrieving Unit(%s) from Registry: %v", name, err))
	}

	// check if the unit is running
	if unit == nil {
		return true, nil
	}
	return false, nil
}

func (f *FleetTunnel) lazyStartUnits(args []string) ([]*schema.Unit, error) {
	units := make([]string, 0, len(args))
	for _, j := range args {
		units = append(units, path.Base(j))
	}
	return f.setTargetStateOfUnits(units, job.JobStateLaunched)
}

// getUnitFile attempts to get a UnitFile configuration
// It takes a unit file name as a parameter and tries first to lookup
// the unit from the local disk. If it fails, it checks if the provided
// file name may reference an instance of a template unit, if so, it
// tries to get the template configuration either from the registry or
// the local disk.
// It returns a UnitFile configuration or nil; and any error ecountered
func getUnitFile(file string) (*unit.UnitFile, error) {
	var uf *unit.UnitFile
	name := path.Base(file)

	log.Debugf("Looking for Unit(%s) or its corresponding template", name)

	// Assume that the file references a local unit file on disk and
	// attempt to load it, if it exists
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		uf, err = getUnitFromFile(file)
		if err != nil {
			return nil, maskAny(fmt.Errorf("failed getting Unit(%s) from file: %v", file, err))
		}
	} else {
		// Otherwise (if the unit file does not exist), check if the
		// name appears to be an instance of a template unit
		info := unit.NewUnitNameInfo(name)
		if info == nil {
			return nil, maskAny(fmt.Errorf("error extracting information from unit name %s", name))
		} else if !info.IsInstance() {
			return nil, maskAny(fmt.Errorf("unable to find Unit(%s) in Registry or on filesystem", name))
		}

		// If it is an instance check for a corresponding template
		// unit in the Registry or disk.
		// If we found a template unit, later we create a
		// near-identical instance unit in the Registry - same
		// unit file as the template, but different name
		uf, err = getUnitFileFromTemplate(info, file)
		if err != nil {
			return nil, maskAny(fmt.Errorf("failed getting Unit(%s) from template: %v", file, err))
		}
	}

	log.Debugf("Found Unit(%s)", name)
	return uf, nil
}

// getUnitFromFile attempts to load a Unit from a given filename
// It returns the Unit or nil, and any error encountered
func getUnitFromFile(file string) (*unit.UnitFile, error) {
	out, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, maskAny(err)
	}

	unitName := path.Base(file)
	log.Debugf("Unit(%s) found in local filesystem", unitName)

	return unit.NewUnitFile(string(out))
}

// getUnitFileFromTemplate attempts to get a Unit from a template unit that
// is either in the registry or on the file system
// It takes two arguments, the template information and the unit file name
// It returns the Unit or nil; and any error encountered
func getUnitFileFromTemplate(uni *unit.UnitNameInfo, fileName string) (*unit.UnitFile, error) {
	// Load template from disk
	filePath := path.Join(path.Dir(fileName), uni.Template)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, maskAny(fmt.Errorf("unable to find template Unit(%s) in Registry or on filesystem", uni.Template))
	}

	uf, err := getUnitFromFile(filePath)
	if err != nil {
		return nil, maskAny(fmt.Errorf("unable to load template Unit(%s) from file: %v", uni.Template, err))
	}

	return uf, nil
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
	err := f.cAPI.CreateUnit(&u)
	if err != nil {
		return nil, maskAny(fmt.Errorf("failed creating unit %s: %v", name, err))
	}

	log.Debugf("Created Unit(%s) in Registry", name)
	return &u, nil
}
