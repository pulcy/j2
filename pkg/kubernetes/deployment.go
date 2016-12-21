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
	"strings"
	"time"

	k8s "github.com/YakLabs/k8s-client"
)

const (
	deploymentStartTimeout = time.Minute
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

// GetCurrent loads the current version of the object on the cluster
func (ds *Deployment) GetCurrent(cs k8s.Client) (interface{}, error) {
	x, err := cs.GetDeployment(ds.Namespace(), ds.Name())
	if err != nil {
		return nil, maskAny(err)
	}
	return &Deployment{*x}, nil
}

// IsEqual returns true of all values configured in myself are the same in the other object.
func (ds *Deployment) IsEqual(other interface{}) ([]string, bool, error) {
	ods, ok := other.(*Deployment)
	if !ok {
		return nil, false, maskAny(fmt.Errorf("Expected other to by *Deployment"))
	}
	if diffs, eq := isSameObjectMeta(ds.Deployment.ObjectMeta, ods.Deployment.ObjectMeta); !eq {
		return diffs, false, nil
	}
	diffs, eq := isSameDeploymentSpec(ds.Spec, ods.Spec)
	return diffs, eq, nil
}

func isSameDeploymentSpec(self, other *k8s.DeploymentSpec) ([]string, bool) {
	if diffs, eq := isSamePodTemplateSpec(&self.Template, &other.Template); !eq {
		return diffs, eq
	}
	diffs, eq := diff(self, other, func(path string) bool {
		switch path {
		case ".Selector":
			return true
		}
		if strings.HasPrefix(path, ".Template") {
			return true
		}
		return false
	})
	return diffs, eq
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
	var lastGeneration int64
	current, err := cs.GetDeployment(ds.Namespace(), ds.Name())
	if err == nil {
		// Update
		events <- "updating"
		lastGeneration = current.Status.ObservedGeneration
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
	if err := ds.waitUntilStarted(cs, events, lastGeneration, deploymentStartTimeout); err != nil {
		return maskAny(err)
	}
	return nil
}

func (ds *Deployment) waitUntilStarted(cs k8s.Client, events chan string, lastGeneration int64, timeout time.Duration) error {
	state := 0
	start := time.Now()
	events <- "waiting for deployment controller"
	for {
		current, err := cs.GetDeployment(ds.Namespace(), ds.Name())
		if err != nil {
			return maskAny(err)
		}
		status := current.Status
		if status != nil {
			switch state {
			case 0:
				// Wait for deployment to update
				if status.ObservedGeneration != lastGeneration {
					events <- "generation updated"
					state = 1
				}
			case 1:
				if status.UpdatedReplicas == ds.Spec.Replicas {
					events <- "all pods updated"
					state = 2
					continue
				}
			case 2:
				if status.AvailableReplicas == ds.Spec.Replicas {
					events <- "all pods available"
					return nil
				}
				events <- fmt.Sprintf("%d pods available, %d unavailable", status.AvailableReplicas, status.UnavailableReplicas)
			}
		}
		if time.Since(start) > timeout {
			return maskAny(fmt.Errorf("Timeout expired"))
		}
		time.Sleep(time.Second * 2)
	}
}
