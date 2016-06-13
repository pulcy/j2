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
	"fmt"

	"github.com/coreos/fleet/schema"
)

func (f *FleetTunnel) Cat(unitName string) (string, error) {
	log.Debugf("cat unit %v", unitName)

	u, err := f.cAPI.Unit(unitName)
	if err != nil {
		return "", maskAny(err)
	}
	if u == nil {
		return "", maskAny(fmt.Errorf("Unit %s not found", unitName))
	}

	uf := schema.MapSchemaUnitOptionsToUnitFile(u.Options)

	return uf.String(), nil
}
