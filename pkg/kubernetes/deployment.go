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
	"encoding/json"
	"fmt"

	"github.com/ericchiang/k8s"
	"github.com/ericchiang/k8s/api/v1"
	"github.com/ericchiang/k8s/apis/extensions/v1beta1"
)

// Deployment is a wrapper for a kubernetes v1beta1.Deployment that implements
// scheduler.UnitData.
type Deployment struct {
	v1beta1.Deployment
}

// Name returns a name of the resource
func (ds *Deployment) Name() string {
	return ds.Deployment.GetMetadata().GetName()
}

// Namespace returns the namespace the resource should be added to.
func (ds *Deployment) Namespace() string {
	return ds.Deployment.GetMetadata().GetNamespace()
}

// ObjectMeta returns the ObjectMeta of the resource.
func (ds *Deployment) ObjectMeta() *v1.ObjectMeta {
	return ds.Deployment.GetMetadata()
}

// Content returns a JSON representation of the resource.
func (ds *Deployment) Content() string {
	x := ds.Deployment
	x.Status = nil
	return mustRender(x)
}

// Destroy deletes the deployment from the cluster.
func (ds *Deployment) Destroy(cs *k8s.Client, events chan string) error {
	ctx := k8s.NamespaceContext(context.Background(), ds.Namespace())
	api := cs.ExtensionsV1Beta1()
	// Fetch current deployment
	current, err := api.GetDeployment(ctx, ds.Name())
	if err != nil {
		return maskAny(err)
	}
	labelSelector := createLabelSelector(current.GetMetadata())

	// Delete deployment itself
	events <- "deleting deployment"
	if err := api.DeleteDeployment(ctx, ds.Name()); err != nil {
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
func (ds *Deployment) Start(cs *k8s.Client, events chan string) error {
	ctx := k8s.NamespaceContext(context.Background(), ds.Namespace())
	api := cs.ExtensionsV1Beta1()
	current, err := api.GetDeployment(ctx, ds.Name())
	if err == nil {
		// Update
		events <- "updating"
		updateMetadataFromCurrent(ds.Deployment.GetMetadata(), current.GetMetadata())
		if _, err := api.UpdateDeployment(ctx, &ds.Deployment); err != nil {
			m, _ := json.Marshal(err)
			fmt.Printf("Error=%s\n", string(m))
			return maskAny(err)
		}
	} else {
		// Create
		events <- "creating"
		if _, err := api.CreateDeployment(ctx, &ds.Deployment); err != nil {
			return maskAny(err)
		}
	}
	return nil
}