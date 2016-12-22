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
	"fmt"
	"time"

	k8s "github.com/YakLabs/k8s-client"
)

const (
	daemonSetPodStartTimeout = time.Minute
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

// GetCurrent loads the current version of the object on the cluster
func (ds *DaemonSet) GetCurrent(cs k8s.Client) (interface{}, error) {
	x, err := cs.GetDaemonSet(ds.Namespace(), ds.Name())
	if err != nil {
		return nil, maskAny(err)
	}
	return &DaemonSet{*x}, nil
}

// IsEqual returns true of all values configured in myself are the same in the other object.
func (ds *DaemonSet) IsEqual(other interface{}) ([]string, bool, error) {
	ods, ok := other.(*DaemonSet)
	if !ok {
		return nil, false, maskAny(fmt.Errorf("Expected other to by *DaemonSet"))
	}
	if diffs, eq := isSameObjectMeta(ds.DaemonSet.ObjectMeta, ods.DaemonSet.ObjectMeta); !eq {
		return diffs, false, nil
	}
	diffs, eq := isSameDaemonSetSpec(ds.Spec, ods.Spec)
	return diffs, eq, nil
}

func isSameDaemonSetSpec(self, other *k8s.DaemonSetSpec) ([]string, bool) {
	diffs, eq := diff(self.Selector, other.Selector, func(path string) bool { return false })
	if !eq {
		return diffs, false
	}
	diffs, eq = isSamePodTemplateSpec(&self.Template, &other.Template)
	return diffs, eq
}

// IsValidState returns true if the current state of the resource on the cluster is OK.
func (ds *DaemonSet) IsValidState(cs k8s.Client) (bool, string, error) {
	return true, "", nil
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
		// First fetch all existing pods
		labelSelector := createLabelSelector(*ds.ObjectMeta())
		pods, err := cs.ListPods(ds.Namespace(), &k8s.ListOptions{LabelSelector: k8s.LabelSelector{MatchLabels: labelSelector}})
		if err != nil {
			return maskAny(err)
		}
		events <- "updating"
		updateMetadataFromCurrent(ds.ObjectMeta(), current.ObjectMeta)
		if _, err := cs.UpdateDaemonSet(ds.Namespace(), &ds.DaemonSet); err != nil {
			return maskAny(err)
		}
		// Delete pods one at a time
		if err := rotatePods(cs, events, pods.Items, labelSelector); err != nil {
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

// rotatePods deletes all given pods 1 at a time.
// For each it that is deleted, it waits until the DaemonSet controller has created and started a new pod.
func rotatePods(cs k8s.Client, events chan string, pods []k8s.Pod, labelSelector map[string]string) error {
	for podIndex, pod := range pods {
		events <- fmt.Sprintf("rotating %s on %s", pod.Name, pod.Status.HostIP)
		if err := cs.DeletePod(pod.Namespace, pod.Name); err != nil {
			return maskAny(err)
		}
		isNewPodRunning := false
		start := time.Now()
		for !isNewPodRunning {
			list, err := cs.ListPods(pod.Namespace, &k8s.ListOptions{LabelSelector: k8s.LabelSelector{MatchLabels: labelSelector}})
			if err != nil {
				return maskAny(err)
			}
			for _, newPod := range list.Items {
				if newPod.Status.HostIP != pod.Status.HostIP || newPod.Name == pod.Name {
					continue
				}
				switch newPod.Status.Phase {
				case k8s.PodPending:
					events <- fmt.Sprintf("pod (%d/%d) %s on %s is pending", podIndex+1, len(pods), newPod.Name, newPod.Status.HostIP)
				case k8s.PodRunning:
					isNewPodRunning = true
					events <- fmt.Sprintf("pod (%d/%d) %s on %s is running", podIndex+1, len(pods), newPod.Name, newPod.Status.HostIP)
				case k8s.PodSucceeded:
					isNewPodRunning = true
					events <- fmt.Sprintf("pod (%d/%d) %s on %s has finished", podIndex+1, len(pods), newPod.Name, newPod.Status.HostIP)
				case k8s.PodFailed:
					return maskAny(fmt.Errorf("Pod %s failed: %s", newPod.Name, newPod.Status.Message))
				default:
				}
			}
			if time.Since(start) > daemonSetPodStartTimeout {
				return maskAny(fmt.Errorf("Pod start timeout on %s", pod.Status.HostIP))
			}
			time.Sleep(time.Second)
		}
		// Wait a bit more
		time.Sleep(time.Second * 5)
	}
	return nil
}
