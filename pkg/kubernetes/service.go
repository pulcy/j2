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

import k8s "github.com/YakLabs/k8s-client"

// Service is a wrapper for a kubernetes v1.Service that implements
// scheduler.UnitData.
type Service struct {
	k8s.Service
}

// Name returns a name of the resource
func (ds *Service) Name() string {
	return ds.Service.ObjectMeta.Name
}

// Namespace returns the namespace the resource should be added to.
func (ds *Service) Namespace() string {
	return ds.Service.ObjectMeta.Namespace
}

// ObjectMeta returns the ObjectMeta of the resource.
func (ds *Service) ObjectMeta() *k8s.ObjectMeta {
	return &ds.Service.ObjectMeta
}

// Content returns a JSON representation of the resource.
func (ds *Service) Content() string {
	x := ds.Service
	x.Status = nil
	return mustRender(x)
}

// Destroy deletes the service from the cluster.
func (ds *Service) Destroy(cs k8s.Client, events chan string) error {
	return maskAny(cs.DeleteService(ds.Namespace(), ds.Name()))
}

// Start creates/updates the service
func (ds *Service) Start(cs k8s.Client, events chan string) error {
	current, err := cs.GetService(ds.Namespace(), ds.Name())
	if err == nil {
		// Update
		events <- "updating"
		updateMetadataFromCurrent(ds.ObjectMeta(), current.ObjectMeta)
		ds.Service.Spec.ClusterIP = current.Spec.ClusterIP
		if _, err := cs.UpdateService(ds.Namespace(), &ds.Service); err != nil {
			return maskAny(err)
		}
	} else {
		// Create
		events <- "creating"
		if _, err := cs.CreateService(ds.Namespace(), &ds.Service); err != nil {
			return maskAny(err)
		}
	}
	return nil
}
