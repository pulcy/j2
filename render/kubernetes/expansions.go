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

package kubernetes

import (
	"fmt"
	"strings"

	"github.com/pulcy/j2/jobs"
	pkg "github.com/pulcy/j2/pkg/kubernetes"
)

const (
	etcdPort = "2379"
)

var (
	etcdHosts = []string{
		"etcd0-etcd0-srv.base",
		"etcd1-etcd1-srv.base",
		"etcd2-etcd2-srv.base",
	}
)

// TODO update for kubernetes

// Expand  "${private_ipv4}":
func (g *k8sRenderer) ExpandPrivateIPv4() string { return "${COREOS_PRIVATE_IPV4}" }

// Expand  "${public_ipv4}":
func (g *k8sRenderer) ExpandPublicIPv4() string { return "${COREOS_PUBLIC_IPV4}" }

// Expand  "${etcd_endpoints}":
func (g *k8sRenderer) ExpandEtcdEndpoints() string {
	eps := make([]string, 0, len(etcdHosts))
	for _, h := range etcdHosts {
		eps = append(eps, fmt.Sprintf("http://%s:%s", h, etcdPort))
	}
	return strings.Join(eps, ",")
}

// Expand  "${etcd_host}":
func (g *k8sRenderer) ExpandEtcdHost() string {
	return etcdHosts[0]
}

// Expand  "${etcd_port}":
func (g *k8sRenderer) ExpandEtcdPort() string { return etcdPort }

// Expand  "${hostname}":
func (g *k8sRenderer) ExpandHostname() string {
	return fmt.Sprintf("${%s}", pkg.EnvVarNodeName)
}

// Expand  "${machine_id}":
func (g *k8sRenderer) ExpandMachineID() string {
	return fmt.Sprintf("${%s}", pkg.EnvVarNodeName) /* as close as we're going to get*/
}

// Expand  "${instance}":
func (g *k8sRenderer) ExpandInstance() string { return "%i" }

// Does the given task support a DNS name link to the given target?
func (g *k8sRenderer) SupportsDNSLinkTo(task *jobs.Task, target jobs.LinkName) bool {
	return !target.HasInstance()
}

// Does the given task support to be linked to itself through a DNS name?
func (g *k8sRenderer) TaskAcceptsDNSLink(task *jobs.Task) bool {
	return task.Type.IsService()
}

// Does the given dependency support to be linked to itself through a DNS name?
func (g *k8sRenderer) DependencyAcceptsDNSLink(d jobs.Dependency) bool {
	return true
}

// TaskDNSName returns the DNS name of the given task
func (g *k8sRenderer) TaskDNSName(task *jobs.Task) string {
	return taskServiceName(task)
}

// DependencyDNSName returns the DNS name used to reach the given dependency
func (g *k8sRenderer) DependencyDNSName(d jobs.Dependency) string {
	return dependencyServiceName(d)
}
