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
)

// deleteReplicaSets deletes the replicateSets that match the given selector from the cluster.
func deleteReplicaSets(cs *kubernetes.Clientset, namespace, labelSelector string) error {
	api := cs.ReplicaSets(namespace)
	if err := api.DeleteCollection(createDeleteOptions(), v1.ListOptions{
		LabelSelector: labelSelector,
	}); err != nil {
		return maskAny(err)
	}
	return nil
}
