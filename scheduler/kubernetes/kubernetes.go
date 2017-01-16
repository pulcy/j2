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
	"github.com/pulcy/j2/cluster"
	"github.com/pulcy/j2/jobs"
	pkg "github.com/pulcy/j2/pkg/kubernetes"
	"github.com/pulcy/j2/scheduler"
	"golang.org/x/sync/errgroup"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

// Unit extends scheduler.Unit with methods used to start & stop units.
type Unit interface {
	scheduler.UnitData
	ObjectMeta() *k8s.ObjectMeta
	Namespace() string
	GetCurrent(cs k8s.Client) (interface{}, error)
	IsEqual(interface{}) ([]string, bool, error)
	IsValidState(cs k8s.Client) (bool, string, error)
	Start(cs k8s.Client, events chan string) error
	Destroy(cs k8s.Client, events chan string) error
}

// NewScheduler creates a new kubernetes implementation of scheduler.Scheduler.
func NewScheduler(j jobs.Job, cluster cluster.Cluster, kubeConfig, context string) (scheduler.Scheduler, error) {
	// creates the client
	client, err := createClientFromConfig(kubeConfig, context)
	if err != nil {
		return nil, maskAny(err)
	}
	return &k8sScheduler{
		client:           client,
		defaultNamespace: pkg.ResourceName(j.Name.String()),
		job:              j,
		cluster:          cluster,
	}, nil
}

type k8sScheduler struct {
	client           k8s.Client
	defaultNamespace string
	job              jobs.Job
	cluster          cluster.Cluster
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
	if list, err := s.listJobs(); err != nil {
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
	ku, ok := unit.(Unit)
	if !ok {
		return scheduler.UnitState{}, maskAny(fmt.Errorf("Expected unit '%s' to implement Kubernetes.Unit", unit.Name()))
	}
	ok, msg, err := ku.IsValidState(s.client)
	if err != nil {
		return scheduler.UnitState{}, maskAny(err)
	}
	state := scheduler.UnitState{
		Failed:  !ok,
		Message: msg,
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

// HasChanged returns true when the given unit is different on the system
func (s *k8sScheduler) HasChanged(unit scheduler.UnitData) ([]string, bool, error) {
	//fmt.Fprintf(os.Stderr, "HasChanged(%s)\n", unit.Name())
	ku, ok := unit.(Unit)
	if !ok {
		return nil, false, maskAny(fmt.Errorf("Expected unit '%s' to be of type Unit", unit.Name()))
	}
	current, err := ku.GetCurrent(s.client)
	if err != nil {
		return nil, false, maskAny(err)
	}
	diffs, eq, err := ku.IsEqual(current)
	if err != nil {
		return nil, false, maskAny(err)
	}
	return diffs, !eq, nil
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
	g := errgroup.Group{}
	for _, u := range units {
		u := u // So we can use u in a inner func
		ku, ok := u.(Unit)
		if !ok {
			return maskAny(fmt.Errorf("Expected unit '%s' to implement KubernetesUnit", u.Name()))
		}
		g.Go(func() error {
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
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return maskAny(err)
	}
	return nil
}

// ensureNamespace checks that the given namespace exists.
// If that is not the case, it is created.
func (s *k8sScheduler) ensureNamespace(namespace string) error {
	// Ensure namespace exists
	nsAPI := s.client
	if _, err := nsAPI.GetNamespace(namespace); err != nil {
		if _, err := nsAPI.CreateNamespace(k8s.NewNamespace(namespace)); err != nil {
			return maskAny(err)
		}
	}
	return nil
}

func (s *k8sScheduler) Start(events chan scheduler.Event, units scheduler.UnitDataList) error {
	g := errgroup.Group{}
	for i := 0; i < units.Len(); i++ {
		unit := units.Get(i)
		ku, ok := unit.(Unit)
		if !ok {
			return maskAny(fmt.Errorf("Expected unit '%s' to implement KubernetesResource", unit.Name()))
		}

		g.Go(func() error {

			// Ensure namespace exists
			if err := s.ensureNamespace(ku.Namespace()); err != nil {
				return maskAny(err)
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
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return maskAny(err)
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
