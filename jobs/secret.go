package jobs

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/juju/errgo"
)

const (
	secretTargetSchemeEnvironment = "environment"
	secretTargetSchemeFile        = "file"
)

// Secret contains a specification of a secret that is to be used by the task.
type Secret struct {
	Path   string `json:"path"`
	Target string `json:"target" mapstructure:"target"`
	Field  string `json:"field,omitempty" mapstructure:"field,omitempty"`
}

// Validate checks the values of the given secret.
// If ok, return nil, otherwise returns an error.
func (s *Secret) Validate() error {
	if s.Path == "" {
		return maskAny(errgo.WithCausef(nil, ValidationError, "path is empty"))
	}
	if s.Target == "" {
		return maskAny(errgo.WithCausef(nil, ValidationError, "target is empty"))
	}
	url, err := url.Parse(s.Target)
	if err != nil {
		return maskAny(errgo.WithCausef(err, ValidationError, "target '%s' is invalid", s.Target))
	}
	switch url.Scheme {
	case secretTargetSchemeEnvironment, secretTargetSchemeFile:
		if url.Host != "" || url.Path == "" || url.Path == "/" || url.RawQuery != "" {
			return maskAny(errgo.WithCausef(err, ValidationError, "target '%s' is invalid", s.Target))
		}
	default:
		return maskAny(errgo.WithCausef(nil, ValidationError, "invalid target scheme '%s'", url.Scheme))
	}
	return nil
}

// TargetEnviroment returns true if the target is an environment variable and if so, the name of the variable.
func (s Secret) TargetEnviroment() (bool, string) {
	url, err := url.Parse(s.Target)
	if err != nil {
		return false, ""
	}
	key := url.Path[1:]
	return url.Scheme == secretTargetSchemeEnvironment, key
}

// TargetFile returns true if the target is a file and if so, the path of the file.
func (s Secret) TargetFile() (bool, string) {
	url, err := url.Parse(s.Target)
	if err != nil {
		return false, ""
	}
	path := url.Path
	return url.Scheme == secretTargetSchemeFile, path
}

// hash returns a has of the given secret config
func (s Secret) hash() (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", maskAny(err)
	}
	hash := sha1.Sum(data)
	return fmt.Sprintf("%x", hash), nil
}