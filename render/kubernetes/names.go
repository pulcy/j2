package kubernetes

import "strings"

const (
	// resource kinds
	kindDeployment = "-dp"
	kindDaemonSet  = "-ds"
	kindPod        = "-pod"
	kindVolume     = "-vol"
)

var (
	resourceNameReplacer = strings.NewReplacer(
		"/", "-",
		"_", "-",
	)
)

// resourceName returns the name of kubernetes resource for the task/group with given fullname.
func resourceName(fullName string, kind string) string {
	prefix := resourceNameReplacer.Replace(fullName)
	return prefix + kind
}
