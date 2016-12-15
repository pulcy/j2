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

// DaemonSet is a wrapper for a kubernetes v1beta1.DaemonSet that implements
// scheduler.UnitData.
type DaemonSet struct {
	v1beta1.DaemonSet
}

// Name returns a name of the resource
func (ds *DaemonSet) Name() string {
	return ds.DaemonSet.Name
}

// Namespace returns the namespace the resource should be added to.
func (ds *DaemonSet) Namespace() string {
	return ds.DaemonSet.ObjectMeta.Namespace
}

// ObjectMeta returns the ObjectMeta of the resource.
func (ds *DaemonSet) ObjectMeta() v1.ObjectMeta {
	return ds.DaemonSet.ObjectMeta
}

// Content returns a JSON representation of the resource.
func (ds *DaemonSet) Content() string {
	x := ds.DaemonSet
	x.Status.Reset()
	return mustRender(x)
}

// Destroy deletes the daemonset from the cluster.
func (ds *DaemonSet) Destroy(cs *kubernetes.Clientset, events chan string) error {
	api := cs.DaemonSets(ds.DaemonSet.Namespace)
	return maskAny(api.Delete(ds.DaemonSet.Name, createDeleteOptions()))
}

// Start creates/updates the daemonSet
func (ds *DaemonSet) Start(cs *kubernetes.Clientset, events chan string) error {
	api := cs.DaemonSets(ds.Namespace())
	current, err := api.Get(ds.DaemonSet.ObjectMeta.Name, metav1.GetOptions{})
	if err == nil {
		// Update
		events <- "updating"
		ds.DaemonSet.ObjectMeta.ResourceVersion = current.ObjectMeta.ResourceVersion
		if _, err := api.Update(&ds.DaemonSet); err != nil {
			return maskAny(err)
		}
	} else {
		// Create
		events <- "creating"
		if _, err := api.Create(&ds.DaemonSet); err != nil {
			return maskAny(err)
		}
	}
	return nil
}
