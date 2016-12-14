package kubernetes

import "strings"

const (
	// resource kinds
	kindDeployment = "-dp"
	kindDaemonSet  = "-ds"
	kindPod        = "-pod"
	kindVolume     = "-vol"
)

// resourceName returns the name of kubernetes resource for the task/group with given fullname.
func resourceName(fullName string, kind string) string {
	return strings.Replace(fullName, "/", "-", -1) + kind
}
