package jobs

type FrontEnd struct {
	Domain     string `json:"domain,omitempty" mapstructure:"domain,omitempty"`
	PathPrefix string `json:"path-prefix,omitempty" mapstructure:"path-prefix,omitempty"`
	SslCert    string `json:"ssl-cert,omitempty" mapstructure:"ssl-cert,omitempty"`
}
