package kubernetes

import (
	k8s "github.com/YakLabs/k8s-client"
)

const (
	// label keys
	LabelPrefix            = "j2."
	LabelJobName           = LabelPrefix + "job.name"
	LabelTaskGroupName     = LabelPrefix + "taskgroup.name"
	LabelTaskGroupFullName = LabelPrefix + "taskgroup.fullname"
	LabelPodName           = LabelPrefix + "pod.name"
)

func isSameLabelSelector(self, other *k8s.LabelSelector) ([]string, bool) {
	var selfMap, otherMap map[string]string
	if self != nil {
		selfMap = self.MatchLabels
	}
	if other != nil {
		otherMap = other.MatchLabels
	}
	if selfMap == nil {
		selfMap = make(map[string]string)
	}
	if otherMap == nil {
		otherMap = make(map[string]string)
	}
	diffs, eq := diff(selfMap, otherMap, func(path string) bool { return false })
	return diffs, eq
}
