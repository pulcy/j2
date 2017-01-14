// Copyright (c) 2017 Pulcy.
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

package robin

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/nyarla/go-crypt"
	"github.com/pulcy/robin-api"

	"github.com/pulcy/j2/jobs"
)

var (
	FixedPwhashSalt string // If set, this salt will be used for all pwhash's (only used for testing)
)

type FrontendRecord struct {
	Record         api.FrontendRecord
	Key            string
	ProjectSetting string
}

// CreateFrontEndRecords create registration code for frontends to the given units to be used by the Robin loadbalancer.
func CreateFrontEndRecords(t *jobs.Task, scalingGroup uint, publicOnly bool, serviceName string) ([]FrontendRecord, error) {
	if len(t.PublicFrontEnds) == 0 && len(t.PrivateFrontEnds) == 0 {
		return nil, nil
	}
	targetServiceName := serviceName
	if t.Type == "proxy" {
		targetServiceName = t.Target.EtcdServiceName()
	}
	httpKey := fmt.Sprintf("/pulcy/frontend/%s-%d", serviceName, scalingGroup)
	httpRecord := api.FrontendRecord{
		Service:         targetServiceName,
		HttpCheckPath:   t.HttpCheckPath,
		HttpCheckMethod: t.HttpCheckMethod,
		Sticky:          t.Sticky,
		Backup:          t.Backup,
		Mode:            "", // Defaults to http
	}
	tcpKey := fmt.Sprintf("/pulcy/frontend/%s-%d-tcp", serviceName, scalingGroup)
	tcpRecord := api.FrontendRecord{
		Service:         targetServiceName,
		HttpCheckPath:   t.HttpCheckPath,
		HttpCheckMethod: t.HttpCheckMethod,
		Sticky:          t.Sticky,
		Backup:          t.Backup,
		Mode:            "tcp",
	}
	instanceHttpKey := fmt.Sprintf("/pulcy/frontend/%s-%d-inst", serviceName, scalingGroup)
	instanceHttpRecord := api.FrontendRecord{
		Service:       fmt.Sprintf("%s-%d", targetServiceName, scalingGroup),
		HttpCheckPath: t.HttpCheckPath,
		Sticky:        t.Sticky,
		Backup:        t.Backup,
	}
	instanceTcpKey := fmt.Sprintf("/pulcy/frontend/%s-%d-inst-tcp", serviceName, scalingGroup)
	instanceTcpRecord := api.FrontendRecord{
		Service:       fmt.Sprintf("%s-%d", targetServiceName, scalingGroup),
		HttpCheckPath: t.HttpCheckPath,
		Sticky:        t.Sticky,
		Backup:        t.Backup,
		Mode:          "tcp",
	}
	var rwRules []api.RewriteRule
	if t.Type == "proxy" && t.Rewrite != nil {
		rwRules = append(rwRules, api.RewriteRule{
			PathPrefix:       t.Rewrite.PathPrefix,
			RemovePathPrefix: t.Rewrite.RemovePathPrefix,
			Domain:           t.Rewrite.Domain,
		})
	}

	for _, fr := range t.PublicFrontEnds {
		record := &httpRecord
		if fr.Mode == "tcp" {
			record = &tcpRecord
		}
		selRecord := api.FrontendSelectorRecord{
			Weight:       fr.Weight,
			Domain:       fr.Domain,
			PathPrefix:   fr.PathPrefix,
			SslCert:      fr.SslCert,
			ServicePort:  fr.Port,
			FrontendPort: fr.HostPort,
			RewriteRules: rwRules,
		}
		if err := addUsers(t, &selRecord, fr.Users); err != nil {
			return nil, maskAny(err)
		}
		record.Selectors = append(record.Selectors, selRecord)
	}

	if !publicOnly {
		for _, fr := range t.PrivateFrontEnds {
			record := &httpRecord
			instanceRecord := &instanceHttpRecord
			if fr.Mode == "tcp" {
				record = &tcpRecord
				instanceRecord = &instanceTcpRecord
			}
			selRecord := api.FrontendSelectorRecord{
				Domain:       t.PrivateDomainName(),
				ServicePort:  fr.Port,
				FrontendPort: fr.HostPort,
				Private:      true,
				RewriteRules: rwRules,
			}
			if err := addUsers(t, &selRecord, fr.Users); err != nil {
				return nil, maskAny(err)
			}
			record.Selectors = append(record.Selectors, selRecord)

			if fr.RegisterInstance {
				instanceSelRecord := selRecord
				instanceSelRecord.Domain = t.InstanceSpecificPrivateDomainName(scalingGroup)
				instanceRecord.Selectors = append(instanceRecord.Selectors, instanceSelRecord)
			}
		}
	}

	var records []FrontendRecord

	if len(instanceHttpRecord.Selectors) > 0 {
		records = append(records, FrontendRecord{
			Record:         instanceHttpRecord,
			Key:            instanceHttpKey,
			ProjectSetting: "FrontEndRegistration-i",
		})
	}
	if len(instanceTcpRecord.Selectors) > 0 {
		records = append(records, FrontendRecord{
			Record:         instanceTcpRecord,
			Key:            instanceTcpKey,
			ProjectSetting: "FrontEndRegistration-i-tcp",
		})
	}
	if len(httpRecord.Selectors) > 0 {
		records = append(records, FrontendRecord{
			Record:         httpRecord,
			Key:            httpKey,
			ProjectSetting: "FrontEndRegistration",
		})
	}
	if len(tcpRecord.Selectors) > 0 {
		records = append(records, FrontendRecord{
			Record:         tcpRecord,
			Key:            tcpKey,
			ProjectSetting: "FrontEndRegistration-tcp",
		})
	}

	return records, nil
}

// addUsers adds the given users to the selector record, while encrypting the passwords.
func addUsers(t *jobs.Task, selRecord *api.FrontendSelectorRecord, users []jobs.User) error {
	if len(users) == 0 {
		return nil
	}
	raw, err := json.Marshal(t)
	if err != nil {
		return maskAny(err)
	}
	saltPrefix := fmt.Sprintf("%x", sha256.Sum256(raw))
	for _, u := range users {
		salt := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%s/%s", u.Name, saltPrefix))))
		userRec := api.UserRecord{
			Name:         u.Name,
			PasswordHash: crypt.Crypt(u.Password, salt),
		}
		selRecord.Users = append(selRecord.Users, userRec)
	}
	return nil
}
