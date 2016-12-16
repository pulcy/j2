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
	"github.com/ericchiang/k8s/apis/extensions/v1beta1"
)

// Ingress is a wrapper for a kubernetes v1beta1.Ingress that implements
// scheduler.UnitData.
type Ingress struct {
	v1beta1.Ingress
}

// Name returns a name of the resource
func (ds *Ingress) Name() string {
	return ds.Ingress.GetMetadata().GetName()
}

// Namespace returns the namespace the resource should be added to.
func (ds *Ingress) Namespace() string {
	return ds.Ingress.GetMetadata().GetNamespace()
}

// ObjectMeta returns the ObjectMeta of the resource.
func (ds *Ingress) ObjectMeta() *v1.ObjectMeta {
	return ds.Ingress.GetMetadata()
}

// Content returns a JSON representation of the resource.
func (ds *Ingress) Content() string {
	x := ds.Ingress
	x.Status = nil
	return mustRender(x)
}

// Destroy deletes the ingress from the cluster.
func (ds *Ingress) Destroy(cs *k8s.Client, events chan string) error {
	ctx := k8s.NamespaceContext(context.Background(), ds.Namespace())
	api := cs.ExtensionsV1Beta1()
	return maskAny(api.DeleteIngress(ctx, ds.Name()))
}

// Start creates/updates the ingress
func (ds *Ingress) Start(cs *k8s.Client, events chan string) error {
	ctx := k8s.NamespaceContext(context.Background(), ds.Namespace())
	api := cs.ExtensionsV1Beta1()
	current, err := api.GetIngress(ctx, ds.Name())
	if err == nil {
		// Update
		events <- "updating"
		updateMetadataFromCurrent(ds.Ingress.GetMetadata(), current.GetMetadata())
		if _, err := api.UpdateIngress(ctx, &ds.Ingress); err != nil {
			return maskAny(err)
		}
	} else {
		// Create
		events <- "creating"
		if _, err := api.CreateIngress(ctx, &ds.Ingress); err != nil {
			return maskAny(err)
		}
	}
	return nil
}
