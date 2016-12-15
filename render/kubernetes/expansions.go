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

// TODO update for kubernetes

// Expand  "${private_ipv4}":
func (g *k8sRenderer) ExpandPrivateIPv4() string { return "${COREOS_PRIVATE_IPV4}" }

// Expand  "${public_ipv4}":
func (g *k8sRenderer) ExpandPublicIPv4() string { return "${COREOS_PUBLIC_IPV4}" }

// Expand  "${etcd_endpoints}":
func (g *k8sRenderer) ExpandEtcdEndpoints() string { return "${ETCD_ENDPOINTS}" }

// Expand  "${etcd_host}":
func (g *k8sRenderer) ExpandEtcdHost() string { return "${ETCD_HOST}" }

// Expand  "${etcd_port}":
func (g *k8sRenderer) ExpandEtcdPort() string { return "${ETCD_PORT}" }

// Expand  "${hostname}":
func (g *k8sRenderer) ExpandHostname() string { return "%H" }

// Expand  "${machine_id}":
func (g *k8sRenderer) ExpandMachineID() string { return "%m" }

// Expand  "${instance}":
func (g *k8sRenderer) ExpandInstance() string { return "%i" }
