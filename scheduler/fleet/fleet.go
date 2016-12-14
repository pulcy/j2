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

package fleetscheduler

import (
	"sync"

	"github.com/juju/errgo"

	"github.com/pulcy/j2/pkg/fleet"
	"github.com/pulcy/j2/scheduler"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

func NewScheduler(tunnel string) (scheduler.Scheduler, error) {
	config := fleet.DefaultConfig()
	config.Tunnel = tunnel
	tun, err := fleet.NewTunnel(config)
	if err != nil {
		return nil, maskAny(err)
	}
	return &fleetScheduler{
		tunnel: *tun,
	}, nil
}

type fleetScheduler struct {
	tunnel      fleet.FleetTunnel
	statusMutex sync.Mutex
	status      *fleet.StatusMap
}

type fleetUnit string

func (u fleetUnit) Name() string {
	return string(u)
}

// List returns the names of all units on the cluster
func (s *fleetScheduler) List() ([]scheduler.Unit, error) {
	names, err := s.tunnel.List()
	if err != nil {
		return nil, maskAny(err)
	}
	units := make([]scheduler.Unit, 0, len(names))
	for _, n := range names {
		units = append(units, fleetUnit(n))
	}
	return units, nil
}

func (s *fleetScheduler) GetState(unit scheduler.Unit) (scheduler.UnitState, error) {
	status, err := s.getStatus()
	if err != nil {
		return scheduler.UnitState{}, maskAny(err)
	}
	unitState, found := status.Get(unit.Name())
	if !found {
		return scheduler.UnitState{}, maskAny(scheduler.NotFoundError)
	}
	state := scheduler.UnitState{
		Failed: unitState == "failed",
	}
	return state, nil
}

func (s *fleetScheduler) Cat(unit scheduler.Unit) (string, error) {
	return s.tunnel.Cat(unit.Name())
}

func (s *fleetScheduler) Stop(events chan scheduler.Event, units ...scheduler.Unit) (scheduler.StopStats, error) {
	s.clearStatus()
	stats, err := s.tunnel.Stop(eventWrapper(events), getUnitNames(units)...)
	if err != nil {
		return scheduler.StopStats{}, maskAny(err)
	}
	return scheduler.StopStats{
		StoppedUnits:       stats.StoppedUnits,
		StoppedGlobalUnits: stats.StoppedGlobalUnits,
	}, nil
}

func (s *fleetScheduler) Destroy(events chan scheduler.Event, units ...scheduler.Unit) error {
	s.clearStatus()
	if err := s.tunnel.Destroy(eventWrapper(events), getUnitNames(units)...); err != nil {
		return maskAny(err)
	}
	return nil
}

func getUnitNames(units []scheduler.Unit) []string {
	names := make([]string, 0, len(units))
	for _, u := range units {
		names = append(names, u.Name())
	}
	return names
}

type unitDataWrapper struct {
	units scheduler.UnitDataList
}

func (l *unitDataWrapper) Len() int {
	return l.units.Len()
}

func (l *unitDataWrapper) Get(index int) fleet.UnitData {
	return l.units.Get(index)
}

func (s *fleetScheduler) Start(events chan scheduler.Event, units scheduler.UnitDataList) error {
	s.clearStatus()
	if err := s.tunnel.Start(eventWrapper(events), &unitDataWrapper{units: units}); err != nil {
		return maskAny(err)
	}
	return nil
}

func (s *fleetScheduler) getStatus() (*fleet.StatusMap, error) {
	s.statusMutex.Lock()
	defer s.statusMutex.Unlock()

	if s.status == nil {
		statusMap, err := s.tunnel.Status()
		if err != nil {
			return nil, maskAny(err)
		}
		s.status = &statusMap
	}
	return s.status, nil
}

func (s *fleetScheduler) clearStatus() {
	s.statusMutex.Lock()
	defer s.statusMutex.Unlock()
	s.status = nil
}

func eventWrapper(events chan scheduler.Event) chan fleet.Event {
	fEvents := make(chan fleet.Event)
	go func() {
		for e := range fEvents {
			events <- scheduler.NewEvent(e.UnitName, e.Message)
		}
	}()
	return fEvents
}
