package kubernetes

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	k8s "github.com/YakLabs/k8s-client"
	"github.com/YakLabs/k8s-client/intstr"
	"gopkg.in/d4l3k/messagediff.v1"
)

func FromInt(i int32) intstr.IntOrString {
	return intstr.FromInt(int(i))
}

func FromString(s string) intstr.IntOrString {
	return intstr.FromString(s)
}

func mustRender(resource interface{}) string {
	raw, err := json.Marshal(resource)
	if err != nil {
		panic(fmt.Sprintf("JSON marshal failed: %#v", err))
	}
	return string(raw)
}

func createLabelSelector(meta k8s.ObjectMeta) map[string]string {
	labels := meta.GetLabels()
	return labels
}

func hasLabels(meta k8s.ObjectMeta, labels map[string]string) bool {
	found := meta.GetLabels()
	for k, v := range labels {
		if foundV, ok := found[k]; !ok {
			return false
		} else if foundV != v {
			return false
		}
	}
	return true
}

func updateMetadataFromCurrent(meta *k8s.ObjectMeta, current k8s.ObjectMeta) {
	meta.ResourceVersion = current.ResourceVersion
	//meta.DeletionGracePeriodSeconds = current.GetDeletionGracePeriodSeconds()
}

func isSameObjectMeta(self, other k8s.ObjectMeta, ignoredLabels ...string) ([]string, bool) {
	d, eq := diff(self, other, func(path string) bool {
		if strings.HasPrefix(path, ".Annotations[") && !strings.Contains(path, "pulcy") {
			return true
		}
		switch path {
		case ".Annotations", ".CreationTimestamp", ".Generation", ".ResourceVersion", ".SelfLink", ".UID":
			return true
		}
		for _, l := range ignoredLabels {
			lPath := fmt.Sprintf(`.Labels["%s"]`, l)
			if lPath == path {
				return true
			}
		}
		return false
	})
	return d, eq
}

// diff compares the given structures.
// If they are equal, (nil, true) is returned.
// If they are different, (a list of diff, false) is returned.
func diff(a, b interface{}, isAllowedModification func(path string) bool) ([]string, bool) {
	d, eq := messagediff.DeepDiff(a, b)
	if eq {
		return nil, true
	}
	if isAllowedModification == nil {
		isAllowedModification = func(string) bool { return false }
	}
	var dstr []string
	for k := range d.Added {
		if !isAllowedModification(k.String()) {
			dstr = append(dstr, fmt.Sprintf("added: %s", k.String()))
		}
	}
	for k := range d.Modified {
		if !isAllowedModification(k.String()) {
			dstr = append(dstr, k.String())
		}
	}
	for k := range d.Removed {
		// Removal is never allowed
		dstr = append(dstr, fmt.Sprintf("removed: %s", k.String()))
	}
	sort.Strings(dstr)
	return dstr, len(dstr) == 0
}
