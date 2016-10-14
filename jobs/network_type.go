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

package jobs

import "github.com/juju/errgo"

// NetworkType is a name of a type of network.
type NetworkType string

const (
	NetworkTypeDefault = NetworkType("default") // Default for engine
	NetworkTypeHost    = NetworkType("host")    // Host network
	NetworkTypeWeave   = NetworkType("weave")   // Weave network
)

// String returns a link name in format <job>.<taskgroup>.<task>
func (nt NetworkType) String() string {
	return string(nt)
}

// Validate returns an error if the given network type is invalid.
// Returns nil on ok.
func (nt NetworkType) Validate() error {
	switch nt {
	case NetworkTypeDefault, NetworkTypeHost, NetworkTypeWeave:
		return nil
	default:
		return maskAny(errgo.WithCausef(nil, ValidationError, "unknown network type '%s'", string(nt)))
	}
}

func (nt NetworkType) IsDefault() bool {
	return nt == NetworkTypeDefault
}

func (nt NetworkType) IsHost() bool {
	return nt == NetworkTypeHost
}

func (nt NetworkType) IsWeave() bool {
	return nt == NetworkTypeWeave
}
