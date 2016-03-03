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
	"github.com/juju/errgo"
)

// PublicFrontEnd contains a specification of a publicly visible HTTP(S) frontend.
type PublicFrontEnd struct {
	Domain     string `json:"domain,omitempty" mapstructure:"domain,omitempty"`
	PathPrefix string `json:"path-prefix,omitempty" mapstructure:"path-prefix,omitempty"`
	SslCert    string `json:"ssl-cert,omitempty" mapstructure:"ssl-cert,omitempty"`
	Port       int    `json:"port,omitempty" mapstructure:"port,omitempty"`
	Users      []User `json:"users,omitempty"`
	Weight     int    `json:"weight,omitempty" mapstructure:"weight,omitempty"`
}

// PrivateFrontEnd contains a specification of a private HTTP(S) frontend.
type PrivateFrontEnd struct {
	Port             int    `json:"port,omitempty" mapstructure:"port,omitempty"`
	Users            []User `json:"users,omitempty"`
	Weight           int    `json:"weight,omitempty" mapstructure:"weight,omitempty"`
	Mode             string `json:"mode,omitempty" mapstructure:"mode,omitempty"`
	RegisterInstance bool   `json:"register-instance,omitempty" mapstructure:"register-instance,omitempty"`
}

// User contains a user name+password who has access to a frontend
type User struct {
	Name     string `json:"name" mapstructure:"name"`
	Password string `json:"password" mapstructure:"password"`
}

func (u User) replaceVariables(ctx *variableContext) User {
	u.Name = ctx.replaceString(u.Name)
	return u
}

func (f PublicFrontEnd) replaceVariables(ctx *variableContext) PublicFrontEnd {
	f.Domain = ctx.replaceString(f.Domain)
	f.PathPrefix = ctx.replaceString(f.PathPrefix)
	f.SslCert = ctx.replaceString(f.SslCert)
	for i, x := range f.Users {
		f.Users[i] = x.replaceVariables(ctx)
	}
	return f
}

// Validate checks the values of the given frontend.
// If ok, return nil, otherwise returns an error.
func (f PublicFrontEnd) Validate() error {
	if f.Weight < 0 || f.Weight > 100 {
		return errgo.WithCausef(nil, ValidationError, "weight must be between 0 and 100")
	}
	if f.SslCert != "" {
		// Domain must be set
		if f.Domain == "" {
			return errgo.WithCausef(nil, ValidationError, "ssl-cert requires a domain setting")
		}
	}
	return nil
}

func (f PrivateFrontEnd) replaceVariables(ctx *variableContext) PrivateFrontEnd {
	f.Mode = ctx.replaceString(f.Mode)
	for i, x := range f.Users {
		f.Users[i] = x.replaceVariables(ctx)
	}
	return f
}

// Validate checks the values of the given frontend.
// If ok, return nil, otherwise returns an error.
func (f PrivateFrontEnd) Validate() error {
	if f.Weight < 0 || f.Weight > 100 {
		return errgo.WithCausef(nil, ValidationError, "weight must be between 0 and 100")
	}
	switch f.Mode {
	case "", "http", "tcp":
		// OK
	default:
		return errgo.WithCausef(nil, ValidationError, "mode must be http or tcp")
	}
	return nil
}

func (f PrivateFrontEnd) IsTcp() bool {
	return f.Mode == "tcp"
}
