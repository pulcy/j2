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
	"encoding/json"
	"fmt"

	k8s "github.com/YakLabs/k8s-client"
)

// Deployment is a wrapper for a kubernetes v1beta1.Deployment that implements
// scheduler.UnitData.
type Deployment struct {
	k8s.Deployment
}

// Name returns a name of the resource
func (ds *Deployment) Name() string {
	return ds.Deployment.ObjectMeta.Name
}

// Namespace returns the namespace the resource should be added to.
func (ds *Deployment) Namespace() string {
	return ds.Deployment.ObjectMeta.Namespace
}

// ObjectMeta returns the ObjectMeta of the resource.
func (ds *Deployment) ObjectMeta() *k8s.ObjectMeta {
	return &ds.Deployment.ObjectMeta
}

// Content returns a JSON representation of the resource.
func (ds *Deployment) Content() string {
	x := ds.Deployment
	x.Status = nil
	return mustRender(x)
}

// Destroy deletes the deployment from the cluster.
func (ds *Deployment) Destroy(cs k8s.Client, events chan string) error {
	// Fetch current deployment
	current, err := cs.GetDeployment(ds.Namespace(), ds.Name())
	if err != nil {
		return maskAny(err)
	}
	labelSelector := createLabelSelector(current.ObjectMeta)

	// Delete deployment itself
	events <- "deleting deployment"
	if err := cs.DeleteDeployment(ds.Namespace(), ds.Name()); err != nil {
		return maskAny(err)
	}

	// Delete created replicaSets.
	events <- "deleting replicaSets"
	if err := deleteReplicaSets(cs, ds.Namespace(), labelSelector); err != nil {
		return maskAny(err)
	}

	// Delete created pods.
	events <- "deleting pods"
	if err := deletePods(cs, ds.Namespace(), labelSelector); err != nil {
		return maskAny(err)
	}
	return nil
}

// Start creates/updates the deployment
func (ds *Deployment) Start(cs k8s.Client, events chan string) error {
	current, err := cs.GetDeployment(ds.Namespace(), ds.Name())
	if err == nil {
		// Update
		events <- "updating"
		updateMetadataFromCurrent(ds.ObjectMeta(), current.ObjectMeta)
		if _, err := cs.UpdateDeployment(ds.Namespace(), &ds.Deployment); err != nil {
			m, _ := json.Marshal(err)
			fmt.Printf("Error=%s\n", string(m))
			return maskAny(err)
		}
	} else {
		// Create
		events <- "creating"
		if _, err := cs.CreateDeployment(ds.Namespace(), &ds.Deployment); err != nil {
			return maskAny(err)
		}
	}
	return nil
}
