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

// Ingress is a wrapper for a kubernetes v1beta1.Ingress that implements
// scheduler.UnitData.
type Ingress struct {
	k8s.Ingress
}

// Name returns a name of the resource
func (ds *Ingress) Name() string {
	return ds.Ingress.ObjectMeta.Name
}

// Namespace returns the namespace the resource should be added to.
func (ds *Ingress) Namespace() string {
	return ds.Ingress.ObjectMeta.Namespace
}

// ObjectMeta returns the ObjectMeta of the resource.
func (ds *Ingress) ObjectMeta() *k8s.ObjectMeta {
	return &ds.Ingress.ObjectMeta
}

// Content returns a JSON representation of the resource.
func (ds *Ingress) Content() string {
	x := ds.Ingress
	x.Status = nil
	return mustRender(x)
}

// Destroy deletes the ingress from the cluster.
func (ds *Ingress) Destroy(cs k8s.Client, events chan string) error {
	return maskAny(cs.DeleteIngress(ds.Namespace(), ds.Name()))
}

// Start creates/updates the ingress
func (ds *Ingress) Start(cs k8s.Client, events chan string) error {
	current, err := cs.GetIngress(ds.Namespace(), ds.Name())
	if err == nil {
		// Update
		events <- "updating"
		updateMetadataFromCurrent(ds.ObjectMeta(), current.ObjectMeta)
		if _, err := cs.UpdateIngress(ds.Namespace(), &ds.Ingress); err != nil {
			return maskAny(err)
		}
	} else {
		// Create
		events <- "creating"
		if _, err := cs.CreateIngress(ds.Namespace(), &ds.Ingress); err != nil {
			return maskAny(err)
		}
	}
	return nil
}
