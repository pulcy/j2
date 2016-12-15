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

	"github.com/pulcy/j2/scheduler"
	"golang.org/x/sync/errgroup"
)

// Destroy removes all unit files that belong to the configured job from the configured cluster.
func (d *Deployment) Destroy() error {
	s, err := d.orchestrator.Scheduler(d.job, d.cluster)
	if err != nil {
		return maskAny(err)
	}

	list, err := s.List()
	if err != nil {
		return maskAny(err)
	}

	predicate := d.createUnitNamePredicate(s)
	unitNames := selectUnitNames(list, predicate)
	if len(unitNames) == 0 {
		fmt.Printf("No units on the cluster match the given arguments\n")
		return nil
	}

	ui := newStateUI(d.verbose)
	defer ui.Close()

	if err := d.confirmDestroy(unitNames, false, ui); err != nil {
		return maskAny(err)
	}
	if err := d.destroyUnits(s, nil, nil, unitNames, ui); err != nil {
		return maskAny(err)
	}

	return nil
}

func (d *Deployment) confirmDestroy(units []scheduler.Unit, obsolete bool, ui *stateUI) error {
	if !d.force {
		obsoleteMsg := ""
		if obsolete {
			obsoleteMsg = " obsolete units"
		}
		names := unitsToNames(units)
		if err := ui.Confirm(fmt.Sprintf("You are about to destroy%s:\n- %s\n\nAre you sure you want to destroy %d units on stack '%s'?\nEnter yes:", obsoleteMsg, strings.Join(names, "\n- "), len(units), d.cluster.Stack)); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

func (d *Deployment) destroyUnits(f scheduler.Scheduler, modifiedUnits, failedUnits, obsoleteUnits []scheduler.Unit, ui *stateUI) error {
	destroy := func(f scheduler.Scheduler, reason scheduler.Reason, units []scheduler.Unit, ui *stateUI) error {
		if len(units) == 0 {
			return nil
		}
		ui.MessageSink <- fmt.Sprintf("Stopping %d unit(s)", len(units))
		stats, err := f.Stop(ui.EventSink, reason, units...)
		if err != nil {
			ui.Warningf("Warning: stop failed.\n%s\n", err.Error())
		}

		if stats.StoppedGlobalUnits > 0 {
			InterruptibleSleep(ui.MessageSink, f.UpdateStopDelay(d.StopDelay), "Waiting for %s...")
		}

		ui.MessageSink <- fmt.Sprintf("Destroying %d unit(s)", len(units))
		if err := f.Destroy(ui.EventSink, reason, units...); err != nil {
			return maskAny(err)
		}
		return nil
	}

	var g errgroup.Group
	g.Go(func() error { return destroy(f, scheduler.ReasonUpdate, modifiedUnits, ui) })
	g.Go(func() error { return destroy(f, scheduler.ReasonFailed, failedUnits, ui) })
	g.Go(func() error { return destroy(f, scheduler.ReasonObsolete, obsoleteUnits, ui) })

	if err := g.Wait(); err != nil {
		return maskAny(err)
	}
	return nil
}

func unitsToNames(units []scheduler.Unit) []string {
	var names []string
	for _, u := range units {
		names = append(names, u.Name())
	}
	return names
}
