package kubernetes

import (
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/client-go/pkg/api/v1"
)

func createDeleteOptions() *v1.DeleteOptions {
	orphanDependents := true
	return &v1.DeleteOptions{
		OrphanDependents: &orphanDependents,
	}
}

func mustRender(resource interface{}) string {
	raw, err := json.Marshal(resource)
	if err != nil {
		panic(fmt.Sprintf("JSON marshal failed: %#v", err))
	}
	return string(raw)
}

func createLabelSelector(meta v1.ObjectMeta) string {
	selector := make([]string, 0, len(meta.Labels))
	for k, v := range meta.Labels {
		selector = append(selector, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(selector, ",")
}
