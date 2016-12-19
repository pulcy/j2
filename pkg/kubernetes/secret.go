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

// Secret is a wrapper for a kubernetes v1.Secret that implements
// scheduler.UnitData.
type Secret struct {
	k8s.Secret
}

// Name returns a name of the resource
func (ds *Secret) Name() string {
	return ds.Secret.ObjectMeta.Name
}

// Namespace returns the namespace the resource should be added to.
func (ds *Secret) Namespace() string {
	return ds.Secret.ObjectMeta.Namespace
}

// ObjectMeta returns the ObjectMeta of the resource.
func (ds *Secret) ObjectMeta() *k8s.ObjectMeta {
	return &ds.Secret.ObjectMeta
}

// Content returns a JSON representation of the resource.
func (ds *Secret) Content() string {
	x := ds.Secret
	return mustRender(x)
}

// Destroy deletes the service from the cluster.
func (ds *Secret) Destroy(cs k8s.Client, events chan string) error {
	return maskAny(cs.DeleteSecret(ds.Namespace(), ds.Name()))
}

// Start creates/updates the secret
func (ds *Secret) Start(cs k8s.Client, events chan string) error {
	current, err := cs.GetSecret(ds.Namespace(), ds.Name())
	if err == nil {
		// Secrets are never updated, unless their labels are different.
		// This is because vault-monkey changes secrets for us and we should not disturb that.
		if !hasLabels(current.ObjectMeta, ds.Secret.ObjectMeta.GetLabels()) {
			// Update
			events <- "updating"
			updateMetadataFromCurrent(ds.ObjectMeta(), current.ObjectMeta)
			if _, err := cs.UpdateSecret(ds.Namespace(), &ds.Secret); err != nil {
				return maskAny(err)
			}
		} else {
			events <- "skip updating"
		}
	} else {
		// Create
		events <- "creating"
		if _, err := cs.CreateSecret(ds.Namespace(), &ds.Secret); err != nil {
			return maskAny(err)
		}
	}
	return nil
}
