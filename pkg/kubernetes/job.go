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
	jobStartTimeout = time.Minute
)

// Job is a wrapper for a kubernetes batch.Job that implements
// scheduler.UnitData.
type Job struct {
	k8s.Job
}

// Name returns a name of the resource
func (ds *Job) Name() string {
	return ds.Job.ObjectMeta.Name
}

// Namespace returns the namespace the resource should be added to.
func (ds *Job) Namespace() string {
	return ds.Job.ObjectMeta.Namespace
}

// GetCurrent loads the current version of the object on the cluster
func (ds *Job) GetCurrent(cs k8s.Client) (interface{}, error) {
	x, err := cs.GetJob(ds.Namespace(), ds.Name())
	if err != nil {
		return nil, maskAny(err)
	}
	return &Job{*x}, nil
}

// IsEqual returns true of all values configured in myself are the same in the other object.
func (ds *Job) IsEqual(other interface{}) ([]string, bool, error) {
	ods, ok := other.(*Job)
	if !ok {
		return nil, false, maskAny(fmt.Errorf("Expected other to by *Job"))
	}
	if diffs, eq := isSameObjectMeta(ds.Job.ObjectMeta, ods.Job.ObjectMeta); !eq {
		return diffs, false, nil
	}
	diffs, eq := isSameJobSpec(ds.Spec, ods.Spec)
	return diffs, eq, nil
}

func isSameJobSpec(self, other *k8s.JobSpec) ([]string, bool) {
	if diffs, eq := isSamePodTemplateSpec(&self.Template, &other.Template, "controller-uid", "job-name"); !eq {
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

// IsValidState returns true if the current state of the resource on the cluster is OK.
func (ds *Job) IsValidState(cs k8s.Client) (bool, string, error) {
	current, err := cs.GetJob(ds.Namespace(), ds.Name())
	if err != nil {
		return false, "", maskAny(err)
	}
	ok := false
	status := current.Status
	msg := ""
	if status != nil {
		ok = status.Failed == 0
		msg = fmt.Sprintf("%d pods active, %d succeeded, %d failed", status.Active, status.Succeeded, status.Failed)
	}
	return ok, msg, nil
}

// ObjectMeta returns the ObjectMeta of the resource.
func (ds *Job) ObjectMeta() *k8s.ObjectMeta {
	return &ds.Job.ObjectMeta
}

// Content returns a JSON representation of the resource.
func (ds *Job) Content() string {
	x := ds.Job
	x.Status = nil
	return mustRender(x)
}

// Destroy deletes the job from the cluster.
func (ds *Job) Destroy(cs k8s.Client, events chan string) error {
	// Fetch current deployment
	current, err := cs.GetJob(ds.Namespace(), ds.Name())
	if err != nil {
		return maskAny(err)
	}
	labelSelector := createLabelSelector(current.ObjectMeta)

	// Delete deployment itself
	events <- "deleting job"
	if err := cs.DeleteJob(ds.Namespace(), ds.Name()); err != nil {
		return maskAny(err)
	}

	// Delete created pods.
	events <- "deleting pods"
	if err := deletePods(cs, ds.Namespace(), labelSelector); err != nil {
		return maskAny(err)
	}
	return nil
}

// Start creates/updates the job
func (ds *Job) Start(cs k8s.Client, events chan string) error {
	current, err := cs.GetJob(ds.Namespace(), ds.Name())
	if err == nil {
		// Update
		events <- "updating"
		updateMetadataFromCurrent(ds.ObjectMeta(), current.ObjectMeta)
		if _, err := cs.UpdateJob(ds.Namespace(), &ds.Job); err != nil {
			m, _ := json.Marshal(err)
			fmt.Printf("Error=%s\n", string(m))
			return maskAny(err)
		}
	} else {
		// Create
		events <- "creating"
		if _, err := cs.CreateJob(ds.Namespace(), &ds.Job); err != nil {
			return maskAny(err)
		}
	}
	return nil
}
