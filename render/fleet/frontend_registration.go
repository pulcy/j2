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
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/pulcy/robin-api"

	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/pkg/robin"
	"github.com/pulcy/j2/pkg/sdunits"
)

// addFrontEndRegistration adds registration code for frontends to the given units
func addFrontEndRegistration(t *jobs.Task, main *sdunits.Unit, ctx generatorContext) error {
	publicOnly := false
	records, err := robin.CreateFrontEndRecords(t, ctx.ScalingGroup, publicOnly, &fleetFrontendNameBuilder{})
	if err != nil {
		return maskAny(err)
	}

	for _, r := range records {
		if err := addFrontEndRegistrationRecord(t, main, r.Key, r.Record, r.ProjectSetting); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

func addFrontEndRegistrationRecord(t *jobs.Task, main *sdunits.Unit, key string, record api.FrontendRecord, projectSettingKey string) error {
	json, err := json.Marshal(&record)
	if err != nil {
		return maskAny(err)
	}
	main.ProjectSetting(projectSettingKey, key+"="+string(json))
	main.ExecOptions.ExecStartPost = append(main.ExecOptions.ExecStartPost,
		fmt.Sprintf("/bin/sh -c 'echo %s | base64 -d | /usr/bin/etcdctl set %s'", base64.StdEncoding.EncodeToString(json), key),
	)
	main.ExecOptions.ExecStop = append(
		[]string{fmt.Sprintf("-/usr/bin/etcdctl rm %s", key)},
		main.ExecOptions.ExecStop...,
	)
	return nil
}

type fleetFrontendNameBuilder struct {
}

// Create the serviceName of the given task.
// This name is used in the Key of the returned records.
func (nb *fleetFrontendNameBuilder) CreateServiceName(t *jobs.Task) (string, error) {
	return t.ServiceName(), nil
}

// Create the name used in the Service field of the returned records.
func (nb *fleetFrontendNameBuilder) CreateTargetServiceName(t *jobs.Task) (string, error) {
	serviceName := t.ServiceName()
	targetServiceName := serviceName
	if t.Type.IsProxy() {
		targetServiceName = t.Target.EtcdServiceName()
	}
	return targetServiceName, nil
}

// Create the Domain field of selectors created for private-frontends.
func (nb *fleetFrontendNameBuilder) CreatePrivateDomainNames(t *jobs.Task) ([]string, error) {
	return []string{t.PrivateDomainName()}, nil
}

// Create the Domain field of selectors created for instance specific private-frontends.
func (nb *fleetFrontendNameBuilder) CreateInstanceSpecificPrivateDomainNames(t *jobs.Task, instance uint) ([]string, error) {
	return []string{t.InstanceSpecificPrivateDomainName(instance)}, nil
}
