package kubernetes

import (
	"fmt"

	"github.com/pulcy/j2/jobs"
	k8s "github.com/pulcy/j2/pkg/kubernetes"
)

const (
	// resource kinds
	kindDeployment = "-depl"
	kindDaemonSet  = "-dset"
	kindIngress    = "-igr"
	kindSecret     = "-sec"
	kindService    = "-srv"
	kindVolume     = "-vol"
)

// resourceName returns the name of kubernetes resource for the task/group with given fullname.
func resourceName(fullName string, kind string) string {
	return k8s.ResourceName(fullName + kind)
}

// taskIngressName creates the name of the ingress created for the given task.
func taskIngressName(t *jobs.Task) string {
	return resourceName(fmt.Sprintf("%s-%s", t.GroupName(), t.Name), kindIngress)
}

// taskServiceName creates the name of the service created for the given task.
func taskServiceName(t *jobs.Task) string {
	return resourceName(fmt.Sprintf("%s-%s", t.GroupName(), t.Name), kindService)
}

// dependencyServiceName creates the name of the service created for the given task.
func dependencyServiceName(d jobs.Dependency) string {
	j, _ := d.Name.Job()
	tg, _ := d.Name.TaskGroup()
	t, _ := d.Name.Task()
	local := resourceName(fmt.Sprintf("%s-%s", tg, t), kindService)
	return resourceName(fmt.Sprintf("%s.%s", local, j), "")
}

// taskSecretName creates the name of the secret created for the given task.
func taskSecretName(t *jobs.Task) string {
	return resourceName(fmt.Sprintf("%s-%s", t.GroupName(), t.Name), kindSecret)
}
