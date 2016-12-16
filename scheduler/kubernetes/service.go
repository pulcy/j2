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
	"context"

	pkg "github.com/pulcy/j2/pkg/kubernetes"
	"github.com/pulcy/j2/scheduler"
)

// listServices returns all services in the namespace
func (s *k8sScheduler) listServices(ctx context.Context) ([]scheduler.Unit, error) {
	var units []scheduler.Unit
	api := s.client.CoreV1()
	if list, err := api.ListServices(ctx); err != nil {
		return nil, maskAny(err)
	} else {
		for _, d := range list.Items {
			units = append(units, &pkg.Service{Service: *d})
		}
	}
	return units, nil
}
