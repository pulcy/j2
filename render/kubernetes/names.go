package kubernetes

import k8s "github.com/pulcy/j2/pkg/kubernetes"

const (
	// resource kinds
	kindDeployment = "-depl"
	kindDaemonSet  = "-dset"
	kindIngress    = "-igr"
	kindService    = "-srv"
	kindVolume     = "-vol"
)

// resourceName returns the name of kubernetes resource for the task/group with given fullname.
func resourceName(fullName string, kind string) string {
	return k8s.ResourceName(fullName + kind)
}
