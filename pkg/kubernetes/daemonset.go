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
	k8s "github.com/YakLabs/k8s-client"
)

// DaemonSet is a wrapper for a kubernetes v1beta1.DaemonSet that implements
// scheduler.UnitData.
type DaemonSet struct {
	k8s.DaemonSet
}

// Name returns a name of the resource
func (ds *DaemonSet) Name() string {
	return ds.DaemonSet.ObjectMeta.Name
}

// Namespace returns the namespace the resource should be added to.
func (ds *DaemonSet) Namespace() string {
	return ds.DaemonSet.ObjectMeta.Namespace
}

// ObjectMeta returns the ObjectMeta of the resource.
func (ds *DaemonSet) ObjectMeta() *k8s.ObjectMeta {
	return &ds.DaemonSet.ObjectMeta
}

// Content returns a JSON representation of the resource.
func (ds *DaemonSet) Content() string {
	x := ds.DaemonSet
	x.Status = nil
	return mustRender(x)
}

// Destroy deletes the daemonset from the cluster.
func (ds *DaemonSet) Destroy(cs k8s.Client, events chan string) error {
	return maskAny(cs.DeleteDaemonSet(ds.Namespace(), ds.Name()))
}

// Start creates/updates the daemonSet
func (ds *DaemonSet) Start(cs k8s.Client, events chan string) error {
	current, err := cs.GetDaemonSet(ds.Namespace(), ds.Name())
	if err == nil {
		// Update
		events <- "updating"
		updateMetadataFromCurrent(ds.ObjectMeta(), current.ObjectMeta)
		if _, err := cs.UpdateDaemonSet(ds.Namespace(), &ds.DaemonSet); err != nil {
			return maskAny(err)
		}
	} else {
		// Create
		events <- "creating"
		if _, err := cs.CreateDaemonSet(ds.Namespace(), &ds.DaemonSet); err != nil {
			return maskAny(err)
		}
	}
	return nil
}
