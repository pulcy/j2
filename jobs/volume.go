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
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/juju/errgo"
)

// VolumeType specifies a type of volume
type VolumeType string

const (
	// VolumeTypeLocal specifies a volume that is mapped onto file system of container host
	VolumeTypeLocal = VolumeType("local")

	// VolumeTypeInstance specifies a volume, managed by j2, that is specific to the task instance
	VolumeTypeInstance = VolumeType("instance")
)

// String returns a volume type as string
func (vt VolumeType) String() string {
	return string(vt)
}

// Validate checks if a volume type follows a valid format
func (vt VolumeType) Validate() error {
	switch string(vt) {
	case "local", "instance", "":
		return nil
	default:
		return maskAny(errgo.WithCausef(nil, ValidationError, "invalid volume type '%s'", string(vt)))
	}
}

// Volume contains a specification of a volume mounted into the tasks container
type Volume struct {
	Path     string     `json:"path"` // container path
	Type     VolumeType `json:"type,omitempty" mapstructure:"type,omitempty"`
	HostPath string     `json:"host-path,omitempty" mapstructure:"host-path,omitempty"`
	Options  []string   `json:"options,omitempty" mapstructure:"options,omitempty"`
}

func (v Volume) replaceVariables(ctx *variableContext) Volume {
	v.Path = ctx.replaceString(v.Path)
	v.Type = VolumeType(ctx.replaceString(string(v.Type)))
	v.HostPath = ctx.replaceString(v.HostPath)
	return v
}

// Validate checks the values of the given volume.
// If ok, return nil, otherwise returns an error.
func (v *Volume) Validate() error {
	if v.Path == "" {
		return maskAny(errgo.WithCausef(nil, ValidationError, "path is empty"))
	}
	if err := v.Type.Validate(); err != nil {
		return maskAny(err)
	}
	return nil
}

// IsLocal returns true of the type of the given volume equals "local"
func (v Volume) IsLocal() bool {
	return v.Type == VolumeTypeLocal
}

// IsInstance returns true of the type of the given volume equals "instance"
func (v Volume) IsInstance() bool {
	return v.Type == VolumeTypeInstance
}

// requiresMountUnit returns true if the type of volume needs a mount unit.
func (v Volume) requiresMountUnit() bool {
	return !v.IsLocal()
}

// PathHash returns a hash of the Path field
func (v Volume) PathHash() string {
	hash := sha1.Sum([]byte(v.Path))
	return fmt.Sprintf("%x", hash[:4])
}

// String creates a string representation of a given volume
func (v Volume) String() string {
	var parts []string
	switch v.Type {
	case VolumeTypeLocal:
		parts = []string{v.HostPath, v.Path}
	case VolumeTypeInstance:
		parts = []string{string(v.Type), v.Path}
	default:
		return ""
	}
	if len(v.Options) > 0 {
		parts = append(parts, strings.Join(v.Options, ","))
	}
	return strings.Join(parts, ":")
}

// MarshalJSON creates a json representation of a given volume
func (v Volume) MarshalJSON() ([]byte, error) {
	str := v.String()
	if str == "" {
		return nil, maskAny(errgo.WithCausef(nil, ValidationError, "invalid type '%s'", v.Type))
	}
	return json.Marshal(str)
}

// ParseVolume parses a string into a Volume
func ParseVolume(input string) (Volume, error) {
	parts := strings.Split(input, ":")
	switch len(parts) {
	case 1:
		return Volume{Type: VolumeTypeInstance, Path: input}, nil
	case 2:
		if VolumeType(parts[0]).Validate() == nil {
			return Volume{Type: VolumeType(parts[0]), Path: parts[1]}, nil
		}
		return Volume{Type: VolumeTypeLocal, Path: parts[1], HostPath: parts[0]}, nil
	case 3:
		if VolumeType(parts[0]).Validate() == nil {
			return Volume{}, maskAny(errgo.WithCausef(nil, ValidationError, "not a valid volume '%s'", input))
		}
		options, err := parseVolumeOptions(parts[2])
		if err != nil {
			return Volume{}, maskAny(err)
		}
		return Volume{Type: VolumeTypeLocal, Path: parts[1], HostPath: parts[0], Options: options}, nil
	default:
		return Volume{}, maskAny(errgo.WithCausef(nil, ValidationError, "not a valid volume '%s'", input))
	}
}

func parseVolumeOptions(input string) ([]string, error) {
	parts := strings.Split(input, ",")
	for _, p := range parts {
		switch p {
		case "rw", "ro", "shared", "private":
			break
		default:
			return nil, maskAny(errgo.WithCausef(nil, ValidationError, "not a valid option '%s'", p))
		}
	}
	return parts, nil
}
