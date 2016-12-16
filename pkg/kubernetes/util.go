package kubernetes

import (
	"encoding/json"
	"fmt"

	"github.com/ericchiang/k8s"
	"github.com/ericchiang/k8s/api/v1"
	"github.com/ericchiang/k8s/util/intstr"
)

func FromInt(i int32) intstr.IntOrString {
	return intstr.IntOrString{
		Type:   k8s.Int64P(0),
		IntVal: k8s.Int32P(i),
	}
}
func FromString(s string) intstr.IntOrString {
	return intstr.IntOrString{
		Type:   k8s.Int64P(1),
		StrVal: k8s.StringP(s),
	}
}

func mustRender(resource interface{}) string {
	raw, err := json.Marshal(resource)
	if err != nil {
		panic(fmt.Sprintf("JSON marshal failed: %#v", err))
	}
	return string(raw)
}

func createLabelSelector(meta *v1.ObjectMeta) map[string]string {
	labels := meta.GetLabels()
	return labels
}

func hasLabels(meta *v1.ObjectMeta, labels map[string]string) bool {
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

func updateMetadataFromCurrent(meta, current *v1.ObjectMeta) {
	meta.ResourceVersion = k8s.StringP(current.GetResourceVersion())
	//meta.DeletionGracePeriodSeconds = current.GetDeletionGracePeriodSeconds()
}
