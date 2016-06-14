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

import (
	"time"

	"github.com/coreos/fleet/registry"
	"github.com/coreos/fleet/ssh"
)

const (
	defaultSSHUserName           = "core"
	defaultSSHTimeout            = time.Duration(10.0) * time.Second
	defaultStrictHostKeyChecking = true
	defaultEndpoint              = "unix:///var/run/fleet.sock"
	defaultRegistryEndpoint      = "http://127.0.0.1:2379,http://127.0.0.1:4001"
	defaultRequestTimeout        = time.Duration(3) * time.Second
	defaultSleepTime             = 500 * time.Millisecond
)

type FleetConfig struct {
	Tunnel                string
	SSHUserName           string
	SSHTimeout            time.Duration
	StrictHostKeyChecking bool
	KnownHostsFile        string
	CAFile                string
	CertFile              string
	KeyFile               string
	EndPoint              string
	RequestTimeout        time.Duration
	EtcdKeyPrefix         string
	NoBlock               bool
	BlockAttempts         int
}

func DefaultConfig() FleetConfig {
	config := FleetConfig{
		Tunnel:                "",
		SSHUserName:           defaultSSHUserName,
		SSHTimeout:            defaultSSHTimeout,
		StrictHostKeyChecking: defaultStrictHostKeyChecking,
		KnownHostsFile:        ssh.DefaultKnownHostsFile,
		CAFile:                "",
		CertFile:              "",
		KeyFile:               "",
		EndPoint:              defaultEndpoint,
		RequestTimeout:        defaultRequestTimeout,
		EtcdKeyPrefix:         registry.DefaultKeyPrefix,
		NoBlock:               false,
		BlockAttempts:         0,
	}
	return config
}
