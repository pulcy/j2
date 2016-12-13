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
	"github.com/juju/errgo"

	"github.com/pulcy/j2/scheduler"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

func NewScheduler(kubeConfig string, namespace string) (scheduler.Scheduler, error) {
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
		namespace: namespace,
	}, nil
}

type k8sScheduler struct {
	clientset *kubernetes.Clientset
	namespace string
}

// List returns the names of all units on the cluster
func (s *k8sScheduler) List() ([]string, error) {
	deployments, err := s.clientset.Deployments(s.namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, maskAny(err)
	}
	names := make([]string, 0, len(deployments.Items))
	for _, d := range deployments.Items {
		names = append(names, d.Name)
	}
	return names, nil
}

func (s *k8sScheduler) GetState(unitName string) (scheduler.UnitState, error) {
	// TODO Implement me
	state := scheduler.UnitState{
		Failed: false,
	}
	return state, nil
}

func (s *k8sScheduler) Cat(unitName string) (string, error) {
	// TODO implement me
	return "", nil
}

func (s *k8sScheduler) Stop(events chan scheduler.Event, unitName ...string) (scheduler.StopStats, error) {
	// TODO implement me
	return scheduler.StopStats{}, nil
}

func (s *k8sScheduler) Destroy(events chan scheduler.Event, unitName ...string) error {
	// TODO implement me
	return nil
}

func (s *k8sScheduler) Start(events chan scheduler.Event, units scheduler.UnitDataList) error {
	// TODO implement me
	return nil
}
