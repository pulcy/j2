package kubernetes

import "strings"

const (
	kindDeployment = "-dp"
	kindDaemonSet  = "-ds"
)

// resourceName returns the name of kubernetes resource for the task/group with given fullname.
func resourceName(fullName string, kind string) string {
	return strings.Replace(fullName, "/", "-", -1) + kind
}
