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
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
)

// Ingress is a wrapper for a kubernetes v1beta1.Ingress that implements
// scheduler.UnitData.
type Ingress struct {
	v1beta1.Ingress
}

// Name returns a name of the resource
func (ds *Ingress) Name() string {
	return ds.Ingress.Name
}

// Namespace returns the namespace the resource should be added to.
func (ds *Ingress) Namespace() string {
	return ds.Ingress.ObjectMeta.Namespace
}

// ObjectMeta returns the ObjectMeta of the resource.
func (ds *Ingress) ObjectMeta() v1.ObjectMeta {
	return ds.Ingress.ObjectMeta
}

// Content returns a JSON representation of the resource.
func (ds *Ingress) Content() string {
	x := ds.Ingress
	x.Status.Reset()
	return mustRender(x)
}

// Destroy deletes the ingress from the cluster.
func (ds *Ingress) Destroy(cs *kubernetes.Clientset, events chan string) error {
	api := cs.Ingresses(ds.Ingress.Namespace)
	return maskAny(api.Delete(ds.Ingress.Name, createDeleteOptions()))
}

// Start creates/updates the ingress
func (ds *Ingress) Start(cs *kubernetes.Clientset, events chan string) error {
	api := cs.Ingresses(ds.Namespace())
	current, err := api.Get(ds.Ingress.ObjectMeta.Name, metav1.GetOptions{})
	if err == nil {
		// Update
		events <- "updating"
		ds.Ingress.ObjectMeta.ResourceVersion = current.ObjectMeta.ResourceVersion
		if _, err := api.Update(&ds.Ingress); err != nil {
			return maskAny(err)
		}
	} else {
		// Create
		events <- "creating"
		if _, err := api.Create(&ds.Ingress); err != nil {
			return maskAny(err)
		}
	}
	return nil
}
