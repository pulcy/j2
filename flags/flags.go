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

package flags

import (
	"time"

	"github.com/pulcy/j2/pkg/vault"
)

type Flags struct {
	// Global flags
	Local          bool // Use local vagrant test cluster
	JobPath        string
	ClusterPath    string
	TunnelOverride string
	Groups         []string
	ScalingGroup   uint
	DryRun         bool
	Force          bool
	AutoContinue   bool
	StopDelay      time.Duration
	DestroyDelay   time.Duration
	SliceDelay     time.Duration
	Options        Options

	vault.VaultConfig
	vault.GithubLoginData
}
