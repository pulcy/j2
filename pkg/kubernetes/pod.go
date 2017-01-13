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

// deletePods deletes the pods that match the given selector from the cluster.
func deletePods(cs k8s.Client, namespace string, labelSelector map[string]string) error {
	all, err := cs.ListPods(namespace, &k8s.ListOptions{LabelSelector: k8s.LabelSelector{MatchLabels: labelSelector}})
	if err != nil {
		return maskAny(err)
	}
	for _, p := range all.Items {
		if !hasLabels(p.ObjectMeta, labelSelector) {
			continue
		}
		if err := cs.DeletePod(namespace, p.ObjectMeta.Name); err != nil {
			return maskAny(err)
		}
	}
	return nil
}

func isSamePodTemplateSpec(self, other *k8s.PodTemplateSpec, ignoredLabels ...string) ([]string, bool) {
	if self == nil {
		return nil, true
	}
	if other == nil {
		return []string{"other=nil"}, false
	}
	if diffs, eq := isSameObjectMeta(self.ObjectMeta, other.ObjectMeta, ignoredLabels...); !eq {
		return diffs, false
	}
	diffs, eq := isSamePodSpec(self.Spec, other.Spec)
	return diffs, eq
}

func isSamePodSpec(self, other *k8s.PodSpec) ([]string, bool) {
	diffs, eq := diff(self, other, func(path string) bool {
		switch path {
		case ".TerminationGracePeriodSeconds":
			return true
		}
		return false
	})
	return diffs, eq
}
