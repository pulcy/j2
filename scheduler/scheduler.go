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

package scheduler

type Scheduler interface {
	// List returns the names of all units on the cluster
	List() ([]string, error)

	GetState(unitName string) (UnitState, error)
	Cat(unitName string) (string, error)

	Stop(events chan Event, unitName ...string) (StopStats, error)
	Destroy(events chan Event, unitName ...string) error

	Start(events chan Event, units UnitDataList) error
}

type UnitState struct {
	Failed bool
}

type StopStats struct {
	StoppedUnits       int
	StoppedGlobalUnits int
}

type Event struct {
	UnitName string
	Message  string
}

type UnitData interface {
	Name() string
	Content() string
}

type UnitDataList interface {
	Len() int
	Get(index int) UnitData
}

func NewEvent(unitName, message string) Event {
	return Event{
		UnitName: unitName,
		Message:  message,
	}
}
