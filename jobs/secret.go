package jobs

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"

	"github.com/juju/errgo"
)

// Secret contains a specification of a secret that is to be used by the task.
type Secret struct {
	Path        string `json:"path"`
	Field       string `json:"field,omitempty" mapstructure:"field,omitempty"`
	Environment string `json:"environment,omitempty" mapstructure:"environment"`
	File        string `json:"file,omitempty" mapstructure:"file"`
}

// Validate checks the values of the given secret.
// If ok, return nil, otherwise returns an error.
func (s *Secret) Validate() error {
	if s.Path == "" {
		return maskAny(errgo.WithCausef(nil, ValidationError, "path is empty"))
	}
	if s.Environment == "" && s.File == "" {
		return maskAny(errgo.WithCausef(nil, ValidationError, "environment and file is empty"))
	}
	return nil
}

// TargetEnviroment returns true if the target is an environment variable and if so, the name of the variable.
func (s Secret) TargetEnviroment() (bool, string) {
	if s.Environment != "" {
		return true, s.Environment
	}
	return false, ""
}

// TargetFile returns true if the target is a file and if so, the path of the file.
func (s Secret) TargetFile() (bool, string) {
	if s.File != "" {
		return true, s.File
	}
	return false, ""
}

// VaultPath returns the path within the vault formatted at <path>[#<field>]
func (s Secret) VaultPath() string {
	path := s.Path
	if s.Field != "" {
		path = path + "#" + s.Field
	}
	return path
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
