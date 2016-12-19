package kubernetes

import (
	"encoding/json"
	"fmt"

	k8s "github.com/YakLabs/k8s-client"
	"github.com/YakLabs/k8s-client/intstr"
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
