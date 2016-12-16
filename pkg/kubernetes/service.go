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
	"context"

	"github.com/ericchiang/k8s"
	"github.com/ericchiang/k8s/api/v1"
)

// Service is a wrapper for a kubernetes v1.Service that implements
// scheduler.UnitData.
type Service struct {
	v1.Service
}

// Name returns a name of the resource
func (ds *Service) Name() string {
	return ds.Service.GetMetadata().GetName()
}

// Namespace returns the namespace the resource should be added to.
func (ds *Service) Namespace() string {
	return ds.Service.GetMetadata().GetNamespace()
}

// ObjectMeta returns the ObjectMeta of the resource.
func (ds *Service) ObjectMeta() *v1.ObjectMeta {
	return ds.Service.GetMetadata()
}

// Content returns a JSON representation of the resource.
func (ds *Service) Content() string {
	x := ds.Service
	x.Status = nil
	return mustRender(x)
}

// Destroy deletes the service from the cluster.
func (ds *Service) Destroy(cs *k8s.Client, events chan string) error {
	ctx := k8s.NamespaceContext(context.Background(), ds.Namespace())
	api := cs.CoreV1()
	return maskAny(api.DeleteService(ctx, ds.Name()))
}

// Start creates/updates the service
func (ds *Service) Start(cs *k8s.Client, events chan string) error {
	ctx := k8s.NamespaceContext(context.Background(), ds.Namespace())
	api := cs.CoreV1()
	current, err := api.GetService(ctx, ds.Name())
	if err == nil {
		// Update
		events <- "updating"
		updateMetadataFromCurrent(ds.Service.GetMetadata(), current.GetMetadata())
		ds.Service.Spec.ClusterIP = current.Spec.ClusterIP
		if _, err := api.UpdateService(ctx, &ds.Service); err != nil {
			return maskAny(err)
		}
	} else {
		// Create
		events <- "creating"
		if _, err := api.CreateService(ctx, &ds.Service); err != nil {
			return maskAny(err)
		}
	}
	return nil
}
