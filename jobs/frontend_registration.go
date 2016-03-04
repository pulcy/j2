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

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/nyarla/go-crypt"

	"github.com/pulcy/j2/units"
)

var (
	FixedPwhashSalt string // If set, this salt will be used for all pwhash's (only used for testing)
)

type frontendRecord struct {
	Selectors     []frontendSelectorRecord `json:"selectors"`
	Service       string                   `json:"service,omitempty"`
	Mode          string                   `json:"mode,omitempty"` // http|tcp
	HttpCheckPath string                   `json:"http-check-path,omitempty"`
	Sticky        bool                     `json:"sticky,omitempty"`
}

type frontendSelectorRecord struct {
	Weight      int          `json:"weight,omitempty"`
	Domain      string       `json:"domain,omitempty"`
	PathPrefix  string       `json:"path-prefix,omitempty"`
	SslCert     string       `json:"ssl-cert,omitempty"`
	Port        int          `json:"port,omitempty"`
	Private     bool         `json:"private,omitempty"`
	Users       []userRecord `json:"users,omitempty"`
	RewriteRule *rewriteRule `json:"rewrite-rule,omitempty"`
}

type userRecord struct {
	Name         string `json:"user"`
	PasswordHash string `json:"pwhash"`
}

type rewriteRule struct {
	PathPrefix string `json:"path-prefix"`
}

// addFrontEndRegistration adds registration code for frontends to the given units
func (t *Task) addFrontEndRegistration(main *units.Unit, ctx generatorContext) error {
	if len(t.PublicFrontEnds) == 0 && len(t.PrivateFrontEnds) == 0 {
		return nil
	}
	serviceName := t.serviceName()
	if t.Type == "proxy" {
		serviceName = t.Target.etcdServiceName()
	}
	key := fmt.Sprintf("/pulcy/frontend/%s-%d", serviceName, ctx.ScalingGroup)
	record := frontendRecord{
		Service:       t.serviceName(),
		HttpCheckPath: t.HttpCheckPath,
		Sticky:        t.Sticky,
	}
	instanceKey := fmt.Sprintf("/pulcy/frontend/%s-%d-inst", serviceName, ctx.ScalingGroup)
	instanceRecord := frontendRecord{
		Service:       fmt.Sprintf("%s-%d", serviceName, ctx.ScalingGroup),
		HttpCheckPath: t.HttpCheckPath,
		Sticky:        t.Sticky,
	}
	var rwRule *rewriteRule
	if t.Type == "proxy" && t.Rewrite != nil {
		rwRule = &rewriteRule{
			PathPrefix: t.Rewrite.PathPrefix,
		}
	}

	for _, fr := range t.PublicFrontEnds {
		selRecord := frontendSelectorRecord{
			Weight:      fr.Weight,
			Domain:      fr.Domain,
			PathPrefix:  fr.PathPrefix,
			SslCert:     fr.SslCert,
			Port:        fr.Port,
			RewriteRule: rwRule,
		}
		if err := selRecord.addUsers(t, fr.Users); err != nil {
			return maskAny(err)
		}
		record.Selectors = append(record.Selectors, selRecord)
	}
	for _, fr := range t.PrivateFrontEnds {
		if fr.Mode == "tcp" {
			record.Mode = "tcp"
		}
		selRecord := frontendSelectorRecord{
			Domain:      t.privateDomainName(),
			Port:        fr.Port,
			Private:     true,
			RewriteRule: rwRule,
		}
		if err := selRecord.addUsers(t, fr.Users); err != nil {
			return maskAny(err)
		}
		record.Selectors = append(record.Selectors, selRecord)

		if fr.RegisterInstance {
			instanceSelRecord := selRecord
			instanceSelRecord.Domain = t.instanceSpecificPrivateDomainName(ctx.ScalingGroup)
			instanceRecord.Selectors = append(instanceRecord.Selectors, instanceSelRecord)
		}
	}

	if len(instanceRecord.Selectors) > 0 {
		if err := t.addFrontEndRegistrationRecord(main, instanceKey, instanceRecord, "FrontEndRegistration-i"); err != nil {
			return maskAny(err)
		}
	}
	if err := t.addFrontEndRegistrationRecord(main, key, record, "FrontEndRegistration"); err != nil {
		return maskAny(err)
	}

	return nil
}

func (t *Task) addFrontEndRegistrationRecord(main *units.Unit, key string, record frontendRecord, projectSettingKey string) error {
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
func (selRecord *frontendSelectorRecord) addUsers(t *Task, users []User) error {
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
		userRec := userRecord{
			Name:         u.Name,
			PasswordHash: crypt.Crypt(u.Password, salt),
		}
		selRecord.Users = append(selRecord.Users, userRec)
	}
	return nil
}
