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

import (
	"time"

	"github.com/pulcy/j2/jobs"
)

type Reason int

const (
	ReasonUpdate   = Reason(1)
	ReasonFailed   = Reason(2)
	ReasonObsolete = Reason(3)
)

type Scheduler interface {
	// ValidateCluster checks if the cluster is suitable to run the configured job.
	ValidateCluster() error

	// ConfigureCluster configures the cluster for use by J2.
	ConfigureCluster(config ClusterConfig) error

	// List returns the names of all units on the cluster
	List() ([]Unit, error)

	GetState(Unit) (UnitState, error)
	//	Cat(Unit) (string, error)

	// HasChanged returns true when the given unit is different on the system (or does not exist on the system)
	HasChanged(UnitData) ([]string, bool, error)

	Stop(events chan Event, reason Reason, units ...Unit) (StopStats, error)
	Destroy(events chan Event, reason Reason, units ...Unit) error

	Start(events chan Event, units UnitDataList) error

	IsUnitForScalingGroup(unit Unit, scalingGroup uint) bool
	IsUnitForJob(unit Unit) bool
	IsUnitForTaskGroup(unit Unit, g jobs.TaskGroupName) bool

	UpdateStopDelay(time.Duration) time.Duration
	UpdateDestroyDelay(time.Duration) time.Duration
}

type ClusterConfig interface {
	ClusterID() string
	VaultConfig
}

type VaultConfig interface {
	VaultAddress() string
	VaultCACert() string
	VaultCAPath() string
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

type Unit interface {
	Name() string
}

type UnitData interface {
	Unit
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
