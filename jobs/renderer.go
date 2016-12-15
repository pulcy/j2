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
}
