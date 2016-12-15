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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
)

// Service is a wrapper for a kubernetes v1.Service that implements
// scheduler.UnitData.
type Service struct {
	v1.Service
}

// Name returns a name of the resource
func (ds *Service) Name() string {
	return ds.Service.Name
}

// Namespace returns the namespace the resource should be added to.
func (ds *Service) Namespace() string {
	return ds.Service.ObjectMeta.Namespace
}

// ObjectMeta returns the ObjectMeta of the resource.
func (ds *Service) ObjectMeta() v1.ObjectMeta {
	return ds.Service.ObjectMeta
}

// Content returns a JSON representation of the resource.
func (ds *Service) Content() string {
	return mustRender(ds.Service)
}

// Destroy deletes the service from the cluster.
func (ds *Service) Destroy(cs *kubernetes.Clientset, events chan string) error {
	api := cs.Services(ds.Service.Namespace)
	return maskAny(api.Delete(ds.Service.Name, createDeleteOptions()))
}

// Start creates/updates the service
func (ds *Service) Start(cs *kubernetes.Clientset, events chan string) error {
	api := cs.Services(ds.Namespace())
	current, err := api.Get(ds.Service.ObjectMeta.Name, metav1.GetOptions{})
	if err == nil {
		// Update
		events <- "updating"
		ds.Service.ObjectMeta.ResourceVersion = current.ObjectMeta.ResourceVersion
		ds.Service.Spec.ClusterIP = current.Spec.ClusterIP
		if _, err := api.Update(&ds.Service); err != nil {
			return maskAny(err)
		}
	} else {
		// Create
		events <- "creating"
		if _, err := api.Create(&ds.Service); err != nil {
			return maskAny(err)
		}
	}
	return nil
}
