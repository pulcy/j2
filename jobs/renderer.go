package jobs

type Renderer interface {
	// Expand  "${private_ipv4}":
	ExpandPrivateIPv4() string

	// Expand  "${public_ipv4}":
	ExpandPublicIPv4() string

	// Expand  "${etcd_endpoints}":
	ExpandEtcdEndpoints() string

	// Expand  "${etcd_host}":
	ExpandEtcdHost() string

	// Expand  "${etcd_port}":
	ExpandEtcdPort() string

	// Expand  "${hostname}":
	ExpandHostname() string

	// Expand  "${machine_id}":
	ExpandMachineID() string

	// Expand  "${instance}":
	ExpandInstance() string

	// Expand  "${kubernetes-namespace}":
	ExpandKubernetesNamespace() string

	// Expand  "${kubernetes-pod}":
	ExpandKubernetesPod() string

	// Does the given task support a DNS name link to the given target?
	SupportsDNSLinkTo(task *Task, target LinkName) bool

	// Does the given task support to be linked to itself through a DNS name?
	TaskAcceptsDNSLink(task *Task) bool

	// Does the given dependency support to be linked to itself through a DNS name?
	DependencyAcceptsDNSLink(d Dependency) bool

	// TaskDNSName returns the DNS name used to reach the given task
	TaskDNSName(task *Task) string

	// DependencyDNSName returns the DNS name used to reach the given dependency
	DependencyDNSName(d Dependency) string
}
