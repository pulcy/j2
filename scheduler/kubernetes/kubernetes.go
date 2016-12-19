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

package kubernetes

import (
	"fmt"
	"time"

	"github.com/juju/errgo"

	k8s "github.com/YakLabs/k8s-client"
	"github.com/pulcy/j2/jobs"
	pkg "github.com/pulcy/j2/pkg/kubernetes"
	"github.com/pulcy/j2/scheduler"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

// Unit extends scheduler.Unit with methods used to start & stop units.
type Unit interface {
	scheduler.UnitData
	ObjectMeta() *k8s.ObjectMeta
	Namespace() string
	Start(cs k8s.Client, events chan string) error
	Destroy(cs k8s.Client, events chan string) error
}

// NewScheduler creates a new kubernetes implementation of scheduler.Scheduler.
func NewScheduler(j jobs.Job, kubeConfig string) (scheduler.Scheduler, error) {
	// creates the client
	client, err := createClientFromConfig(kubeConfig)
	if err != nil {
		return nil, maskAny(err)
	}
	return &k8sScheduler{
		client:           client,
		defaultNamespace: pkg.ResourceName(j.Name.String()),
		job:              j,
	}, nil
}

type k8sScheduler struct {
	client           k8s.Client
	defaultNamespace string
	job              jobs.Job
}

// List returns the names of all units on the cluster
func (s *k8sScheduler) List() ([]scheduler.Unit, error) {
	var units []scheduler.Unit
	if list, err := s.listDeployments(); err != nil {
		return nil, maskAny(err)
	} else {
		units = append(units, list...)
	}
	if list, err := s.listDaemonSets(); err != nil {
		return nil, maskAny(err)
	} else {
		units = append(units, list...)
	}
	if list, err := s.listServices(); err != nil {
		return nil, maskAny(err)
	} else {
		units = append(units, list...)
	}
	if list, err := s.listSecrets(); err != nil {
		return nil, maskAny(err)
	} else {
		units = append(units, list...)
	}
	if list, err := s.listIngresses(); err != nil {
		return nil, maskAny(err)
	} else {
		units = append(units, list...)
	}
	// TODO load other resources
	return units, nil
}

func (s *k8sScheduler) GetState(unit scheduler.Unit) (scheduler.UnitState, error) {
	// TODO Implement me
	state := scheduler.UnitState{
		Failed: false,
	}
	return state, nil
}

func (s *k8sScheduler) Cat(unit scheduler.Unit) (string, error) {
	ku, ok := unit.(Unit)
	if !ok {
		return "", maskAny(fmt.Errorf("Expected unit '%s' to implement Kubernetes.Unit", unit.Name()))
	}
	return ku.Content(), nil
}

func (s *k8sScheduler) Stop(events chan scheduler.Event, reason scheduler.Reason, units ...scheduler.Unit) (scheduler.StopStats, error) {
	return scheduler.StopStats{
		StoppedUnits:       len(units),
		StoppedGlobalUnits: 0,
	}, nil
}

func (s *k8sScheduler) Destroy(events chan scheduler.Event, reason scheduler.Reason, units ...scheduler.Unit) error {
	if reason != scheduler.ReasonObsolete {
		return nil
	}
	for _, u := range units {
		ku, ok := u.(Unit)
		if !ok {
			return maskAny(fmt.Errorf("Expected unit '%s' to implement KubernetesUnit", u.Name()))
		}
		destroyEvents := make(chan string)
		go func() {
			for msg := range destroyEvents {
				events <- scheduler.Event{
					UnitName: u.Name(),
					Message:  msg,
				}
			}
		}()
		if err := ku.Destroy(s.client, destroyEvents); err != nil {
			return maskAny(err)
		}
		close(destroyEvents)
		events <- scheduler.Event{
			UnitName: u.Name(),
			Message:  "destroyed",
		}
	}
	return nil
}

func (s *k8sScheduler) Start(events chan scheduler.Event, units scheduler.UnitDataList) error {
	for i := 0; i < units.Len(); i++ {
		unit := units.Get(i)
		ku, ok := unit.(Unit)
		if !ok {
			return maskAny(fmt.Errorf("Expected unit '%s' to implement KubernetesResource", unit.Name()))
		}

		// Ensure namespace exists
		nsAPI := s.client
		if _, err := nsAPI.GetNamespace(ku.Namespace()); err != nil {
			if _, err := nsAPI.CreateNamespace(k8s.NewNamespace(ku.Namespace())); err != nil {
				return maskAny(err)
			}
		}

		// Create/update resource
		startEvents := make(chan string)
		go func() {
			for msg := range startEvents {
				events <- scheduler.Event{
					UnitName: unit.Name(),
					Message:  msg,
				}
			}
		}()
		if err := ku.Start(s.client, startEvents); err != nil {
			return maskAny(err)
		}
		close(startEvents)
		events <- scheduler.Event{
			UnitName: unit.Name(),
			Message:  "started",
		}
	}
	return nil
}

// IsUnitForScalingGroup returns true if the given unit is part of the job this scheduler was build for.
func (s *k8sScheduler) IsUnitForScalingGroup(unit scheduler.Unit, scalingGroup uint) bool {
	return s.IsUnitForJob(unit)
}

// IsUnitForJob returns true if the given unit is part of the job this scheduler was build for.
func (s *k8sScheduler) IsUnitForJob(unit scheduler.Unit) bool {
	if ku, ok := unit.(Unit); !ok {
		return false
	} else {
		found := ku.ObjectMeta().Labels[pkg.LabelJobName]
		expected := pkg.ResourceName(s.job.Name.String())
		if found != expected {
			return false
		}
		return true
	}
}

// IsUnitForTaskGroup returns true if the given unit is part of the job this scheduler was build for
// and part of the task group with given name.
func (s *k8sScheduler) IsUnitForTaskGroup(unit scheduler.Unit, g jobs.TaskGroupName) bool {
	if !s.IsUnitForJob(unit) {
		return false
	}
	if ku, ok := unit.(Unit); !ok {
		return false
	} else {
		if ku.ObjectMeta().Labels[pkg.LabelTaskGroupName] == pkg.ResourceName(g.String()) {
			return true
		}
		return false
	}
}

func (s *k8sScheduler) UpdateStopDelay(d time.Duration) time.Duration {
	// Stopping is done by Kubernetes, do not wait for it
	return time.Duration(0)
}

func (s *k8sScheduler) UpdateDestroyDelay(d time.Duration) time.Duration {
	// Destroying is done inline by Kubernetes, do not wait for it
	return time.Duration(0)
}
