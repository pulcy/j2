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

// Deployment is a wrapper for a kubernetes v1beta1.Deployment that implements
// scheduler.UnitData.
type Deployment struct {
	v1beta1.Deployment
}

// Name returns a name of the resource
func (ds *Deployment) Name() string {
	return ds.Deployment.Name
}

// Namespace returns the namespace the resource should be added to.
func (ds *Deployment) Namespace() string {
	return ds.Deployment.ObjectMeta.Namespace
}

// ObjectMeta returns the ObjectMeta of the resource.
func (ds *Deployment) ObjectMeta() v1.ObjectMeta {
	return ds.Deployment.ObjectMeta
}

// Content returns a JSON representation of the resource.
func (ds *Deployment) Content() string {
	x := ds.Deployment
	x.Status.Reset()
	return mustRender(x)
}

// Destroy deletes the deployment from the cluster.
func (ds *Deployment) Destroy(cs *kubernetes.Clientset, events chan string) error {
	api := cs.Deployments(ds.Deployment.Namespace)
	// Fetch current deployment
	current, err := api.Get(ds.Deployment.Name, metav1.GetOptions{})
	if err != nil {
		return maskAny(err)
	}
	labelSelector := createLabelSelector(current.ObjectMeta)

	// Delete deployment itself
	events <- "deleting deployment"
	if err := api.Delete(ds.Deployment.Name, createDeleteOptions()); err != nil {
		return maskAny(err)
	}

	// Delete created replicaSets.
	events <- "deleting replicaSets"
	if err := deleteReplicaSets(cs, ds.Deployment.Namespace, labelSelector); err != nil {
		return maskAny(err)
	}

	// Delete created pods.
	events <- "deleting pods"
	if err := deletePods(cs, ds.Deployment.Namespace, labelSelector); err != nil {
		return maskAny(err)
	}
	return nil
}

// Start creates/updates the deployment
func (ds *Deployment) Start(cs *kubernetes.Clientset, events chan string) error {
	api := cs.Deployments(ds.Namespace())
	current, err := api.Get(ds.Deployment.ObjectMeta.Name, metav1.GetOptions{})
	if err == nil {
		// Update
		events <- "updating"
		ds.Deployment.ObjectMeta.ResourceVersion = current.ObjectMeta.ResourceVersion
		if _, err := api.Update(&ds.Deployment); err != nil {
			return maskAny(err)
		}
	} else {
		// Create
		events <- "creating"
		if _, err := api.Create(&ds.Deployment); err != nil {
			return maskAny(err)
		}
	}
	return nil
}
