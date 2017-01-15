// Copyright (c) 2016 Pulcy.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fleet

import "github.com/pulcy/j2/jobs"

// Expand  "${private_ipv4}":
func (g *fleetRenderer) ExpandPrivateIPv4() string { return "${COREOS_PRIVATE_IPV4}" }

// Expand  "${public_ipv4}":
func (g *fleetRenderer) ExpandPublicIPv4() string { return "${COREOS_PUBLIC_IPV4}" }

// Expand  "${etcd_endpoints}":
func (g *fleetRenderer) ExpandEtcdEndpoints() string { return "${ETCD_ENDPOINTS}" }

// Expand  "${etcd_host}":
func (g *fleetRenderer) ExpandEtcdHost() string { return "${ETCD_HOST}" }

// Expand  "${etcd_port}":
func (g *fleetRenderer) ExpandEtcdPort() string { return "${ETCD_PORT}" }

// Expand  "${hostname}":
func (g *fleetRenderer) ExpandHostname() string { return "%H" }

// Expand  "${machine_id}":
func (g *fleetRenderer) ExpandMachineID() string { return "%m" }

// Expand  "${instance}":
func (g *fleetRenderer) ExpandInstance() string { return "%i" }

// Does the given task support a DNS name link to the given target?
func (g *fleetRenderer) SupportsDNSLinkTo(task *jobs.Task, target jobs.LinkName) bool {
	return task.Network.IsWeave() && !target.HasInstance()
}

// Does the given task support to be linked to itself through a DNS name?
func (g *fleetRenderer) TaskAcceptsDNSLink(task *jobs.Task) bool {
	return task.Type.IsService() && task.Network.IsWeave()
}

// Does the given dependency support to be linked to itself through a DNS name?
func (g *fleetRenderer) DependencyAcceptsDNSLink(d jobs.Dependency) bool {
	return d.Network.IsWeave()
}

// TaskDNSName returns the DNS name of the given task
func (g *fleetRenderer) TaskDNSName(task *jobs.Task) string {
	return task.WeaveDomainName()
}

// DependencyDNSName returns the DNS name used to reach the given dependency
func (g *fleetRenderer) DependencyDNSName(d jobs.Dependency) string {
	return d.Name.WeaveDomainName()
}
