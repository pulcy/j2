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
}

// PrivateFrontEnd contains a specification of a private HTTP(S) frontend.
type PrivateFrontEnd struct {
	Port  int    `json:"port,omitempty" mapstructure:"port,omitempty"`
	Users []User `json:"users,omitempty"`
}

// User contains a user name+password who has access to a frontend
type User struct {
	Name     string `json:"name" mapstructure:"name"`
	Password string `json:"password" mapstructure:"password"`
}

// Validate checks the values of the given frontend.
// If ok, return nil, otherwise returns an error.
func (f *PublicFrontEnd) Validate() error {
	if f.SslCert != "" {
		// Domain must be set
		if f.Domain == "" {
			return errgo.WithCausef(nil, ValidationError, "ssl-cert requires a domain setting")
		}
	}
	return nil
}

// Validate checks the values of the given frontend.
// If ok, return nil, otherwise returns an error.
func (f *PrivateFrontEnd) Validate() error {
	return nil
}
