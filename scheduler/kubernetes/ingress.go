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
	"github.com/pulcy/j2/scheduler"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

type ingressUnit struct {
	v1beta1.Ingress
}

func (u *ingressUnit) Name() string {
	return u.Ingress.Name
}

func (u *ingressUnit) Destroy(cs *kubernetes.Clientset) error {
	api := cs.Ingresses(u.Ingress.Namespace)
	return maskAny(api.Delete(u.Ingress.Name, createDeleteOptions()))
}

// listIngresses returns all ingresses in the namespace
func (s *k8sScheduler) listIngresses() ([]scheduler.Unit, error) {
	var units []scheduler.Unit
	if list, err := s.clientset.Ingresses(s.namespace).List(v1.ListOptions{}); err != nil {
		return nil, maskAny(err)
	} else {
		for _, d := range list.Items {
			units = append(units, &ingressUnit{Ingress: d})
		}
	}
	return units, nil
}
