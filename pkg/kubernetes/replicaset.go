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

	"github.com/ericchiang/k8s"
)

// deleteReplicaSets deletes the replicateSets that match the given selector from the cluster.
func deleteReplicaSets(cs *k8s.Client, namespace string, labelSelector map[string]string) error {
	ctx := k8s.NamespaceContext(context.Background(), namespace)
	api := cs.ExtensionsV1Beta1()
	all, err := api.ListReplicaSets(ctx)
	if err != nil {
		return maskAny(err)
	}
	for _, rs := range all.Items {
		if !hasLabels(rs.GetMetadata(), labelSelector) {
			continue
		}
		if err := api.DeleteReplicaSet(ctx, rs.GetMetadata().GetName()); err != nil {
			return maskAny(err)
		}
	}
	return nil
}
