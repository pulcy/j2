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
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/nyarla/go-crypt"
	"github.com/pulcy/robin-api"

	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/pkg/sdunits"
)

var (
	FixedPwhashSalt string // If set, this salt will be used for all pwhash's (only used for testing)
)

// addFrontEndRegistration adds registration code for frontends to the given units
func addFrontEndRegistration(t *jobs.Task, main *sdunits.Unit, ctx generatorContext) error {
	if len(t.PublicFrontEnds) == 0 && len(t.PrivateFrontEnds) == 0 {
		return nil
	}
	serviceName := t.ServiceName()
	targetServiceName := serviceName
	if t.Type == "proxy" {
		targetServiceName = t.Target.EtcdServiceName()
	}
	key := fmt.Sprintf("/pulcy/frontend/%s-%d", serviceName, ctx.ScalingGroup)
	record := api.FrontendRecord{
		Service:         targetServiceName,
		HttpCheckPath:   t.HttpCheckPath,
		HttpCheckMethod: t.HttpCheckMethod,
		Sticky:          t.Sticky,
		Backup:          t.Backup,
	}
	instanceKey := fmt.Sprintf("/pulcy/frontend/%s-%d-inst", serviceName, ctx.ScalingGroup)
	instanceRecord := api.FrontendRecord{
		Service:       fmt.Sprintf("%s-%d", targetServiceName, ctx.ScalingGroup),
		HttpCheckPath: t.HttpCheckPath,
		Sticky:        t.Sticky,
		Backup:        t.Backup,
	}
	var rwRules []api.RewriteRule
	if t.Type == "proxy" && len(t.Rewrites) > 0 {
		for _, rw := range t.Rewrites {
			rwRules = append(rwRules, api.RewriteRule{
				PathPrefix:       rw.PathPrefix,
				RemovePathPrefix: rw.RemovePathPrefix,
				Domain:           rw.Domain,
			})
		}
	}

	for _, fr := range t.PublicFrontEnds {
		selRecord := api.FrontendSelectorRecord{
			Weight:       fr.Weight,
			Domain:       fr.Domain,
			PathPrefix:   fr.PathPrefix,
			SslCert:      fr.SslCert,
			ServicePort:  fr.Port,
			RewriteRules: rwRules,
		}
		if err := addUsers(t, &selRecord, fr.Users); err != nil {
			return maskAny(err)
		}
		record.Selectors = append(record.Selectors, selRecord)
	}
	for _, fr := range t.PrivateFrontEnds {
		if fr.Mode == "tcp" {
			record.Mode = "tcp"
		}
		selRecord := api.FrontendSelectorRecord{
			Domain:       t.PrivateDomainName(),
			ServicePort:  fr.Port,
			Private:      true,
			RewriteRules: rwRules,
		}
		if err := addUsers(t, &selRecord, fr.Users); err != nil {
			return maskAny(err)
		}
		record.Selectors = append(record.Selectors, selRecord)

		if fr.RegisterInstance {
			instanceSelRecord := selRecord
			instanceSelRecord.Domain = t.InstanceSpecificPrivateDomainName(ctx.ScalingGroup)
			instanceRecord.Selectors = append(instanceRecord.Selectors, instanceSelRecord)
		}
	}

	if len(instanceRecord.Selectors) > 0 {
		if err := addFrontEndRegistrationRecord(t, main, instanceKey, instanceRecord, "FrontEndRegistration-i"); err != nil {
			return maskAny(err)
		}
	}
	if err := addFrontEndRegistrationRecord(t, main, key, record, "FrontEndRegistration"); err != nil {
		return maskAny(err)
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
