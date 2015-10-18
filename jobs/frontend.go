package jobs

import (
	"github.com/juju/errgo"
)

type FrontEnd struct {
	Domain      string `json:"domain,omitempty" mapstructure:"domain,omitempty"`
	PathPrefix  string `json:"path-prefix,omitempty" mapstructure:"path-prefix,omitempty"`
	SslCert     string `json:"ssl-cert,omitempty" mapstructure:"ssl-cert,omitempty"`
	PrivatePort int    `json:"private-port,omitempty" mapstructure:"private-port,omitempty"`
}

// Validate checks the values of the given frontend.
// If ok, return nil, otherwise returns an error.
func (f *FrontEnd) Validate() error {
	if f.SslCert != "" {
		// Domain must be set
		if f.Domain == "" {
			return errgo.WithCausef(nil, ValidationError, "ssl-cert requires a domain setting")
		}
	}
	if f.Domain != "" {
		// PrivatePort must not be set
		if f.PrivatePort != 0 {
			return errgo.WithCausef(nil, ValidationError, "domain and private-port cannot be both set")
		}
	}
	return nil
}
