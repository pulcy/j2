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
	kindJob        = "-job"
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

// taskServiceDNSName creates the DNS name of the service created for the given task.
// This allows the service to be reached from other namespaces.
func taskServiceDNSName(t *jobs.Task, clusterDomain string) string {
	domain := k8s.ResourceName(t.JobName().String())
	return fmt.Sprintf("%s.%s.svc.%s", taskServiceName(t), domain, clusterDomain)
}

// dependencyServiceName creates the name of the service created for the given task.
func dependencyServiceName(d jobs.Dependency) string {
	j, _ := d.Name.Job()
	tg, _ := d.Name.TaskGroup()
	t, _ := d.Name.Task()
	local := resourceName(fmt.Sprintf("%s-%s", tg, t), kindService)
	return resourceName(fmt.Sprintf("%s.%s", local, j), "")
}

// dependencyServiceName creates the name of the service created for the given task.
func dependencyServiceDNSName(d jobs.Dependency, clusterDomain string) string {
	local := dependencyServiceName(d)
	return fmt.Sprintf("%s.svc.%s", local, clusterDomain)
}

// taskSecretName creates the name of the secret created for the given task.
func taskSecretName(t *jobs.Task) string {
	return resourceName(fmt.Sprintf("%s-%s", t.GroupName(), t.Name), kindSecret)
}
