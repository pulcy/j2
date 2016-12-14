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

	"github.com/juju/errgo"

	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/scheduler"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

type KubernetesResource interface {
	Namespace() string
	Start(*kubernetes.Clientset) error
}

func NewScheduler(j jobs.Job, kubeConfig string) (scheduler.Scheduler, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		return nil, maskAny(err)
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, maskAny(err)
	}
	return &k8sScheduler{
		clientset: clientset,
	}, nil
}

type k8sScheduler struct {
	clientset *kubernetes.Clientset
}

type deploymentUnit struct {
	v1beta1.Deployment
}

func (u *deploymentUnit) Name() string { return u.Deployment.Name }

// List returns the names of all units on the cluster
func (s *k8sScheduler) List() ([]scheduler.Unit, error) {
	var units []scheduler.Unit
	namespace := ""
	if list, err := s.clientset.Deployments(namespace).List(v1.ListOptions{}); err != nil {
		return nil, maskAny(err)
	} else {
		for _, d := range list.Items {
			units = append(units, &deploymentUnit{Deployment: d})
		}
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
	// TODO implement me
	return "", nil
}

func (s *k8sScheduler) Stop(events chan scheduler.Event, units ...scheduler.Unit) (scheduler.StopStats, error) {
	return scheduler.StopStats{
		StoppedUnits:       len(units),
		StoppedGlobalUnits: 0,
	}, nil
}

func (s *k8sScheduler) Destroy(events chan scheduler.Event, units ...scheduler.Unit) error {
	// TODO implement me
	return nil
}

func (s *k8sScheduler) Start(events chan scheduler.Event, units scheduler.UnitDataList) error {
	for i := 0; i < units.Len(); i++ {
		unit := units.Get(i)
		res, ok := unit.(KubernetesResource)
		if !ok {
			return maskAny(fmt.Errorf("Expected unit '%s' to implement KubernetesResource", unit.Name()))
		}

		// Ensure namespace exists
		nsAPI := s.clientset.Namespaces()
		if _, err := nsAPI.Get(res.Namespace(), metav1.GetOptions{}); err != nil {
			if _, err := nsAPI.Create(createNamespace(res.Namespace())); err != nil {
				return maskAny(err)
			}
		}

		// Create/update resource
		if err := res.Start(s.clientset); err != nil {
			return maskAny(err)
		}
		events <- scheduler.Event{
			UnitName: unit.Name(),
			Message:  "started",
		}
	}
	return nil
}

func createNamespace(name string) *v1.Namespace {
	ns := &v1.Namespace{}
	ns.TypeMeta.Kind = "Namespace"
	ns.ObjectMeta.Name = name
	return ns
}
