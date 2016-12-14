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
	"encoding/json"
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
	Path         string     `json:"path"` // container path
	Type         VolumeType `json:"type,omitempty" mapstructure:"type,omitempty"`
	HostPath     string     `json:"host-path,omitempty" mapstructure:"host-path,omitempty"`
	Options      []string   `json:"options,omitempty" mapstructure:"options,omitempty"`
	MountOptions []string   `json:"mount-options,omitempty" mapstructure:"mount-options,omitempty"`
}

func (v Volume) replaceVariables(ctx *variableContext) Volume {
	v.Path = ctx.replaceString(v.Path)
	v.Type = VolumeType(ctx.replaceString(string(v.Type)))
	v.HostPath = ctx.replaceString(v.HostPath)
	v.Options = ctx.replaceStringSlice(v.Options)
	v.MountOptions = ctx.replaceStringSlice(v.MountOptions)
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

// IsReadOnly returns true if the given volume contains the "ro" option.
func (v Volume) IsReadOnly() bool {
	for _, o := range v.Options {
		if o == "ro" {
			return true
		}
	}
	return false
}

// MountOption looks for a mount option with given key and returns its value.
// Returns OptionNotFoundError if option is not found.
func (v Volume) MountOption(key string) (string, error) {
	for _, x := range v.MountOptions {
		if x == key {
			return "", nil
		}
		if strings.HasPrefix(x, key+"=") {
			return x[len(key)+1:], nil
		}
	}
	return "", maskAny(errgo.WithCausef(nil, OptionNotFoundError, key))
}

// String creates a string representation of a given volume
func (v Volume) String() string {
	var parts []string
	switch v.Type {
	case VolumeTypeLocal:
		parts = []string{v.HostPath, v.Path}
	case VolumeTypeInstance:
		parts = []string{string(v.Type), v.Path}
		if len(v.MountOptions) > 0 {
			parts[0] = parts[0] + "@" + strings.Join(v.MountOptions, ",")
		}
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
		if vt, mountOptions, err := parseVolumeType(parts[0]); err == nil {
			return Volume{Type: vt, Path: parts[1], MountOptions: mountOptions}, nil
		}
		return Volume{Type: VolumeTypeLocal, Path: parts[1], HostPath: parts[0]}, nil
	case 3:
		if _, _, err := parseVolumeType(parts[0]); err == nil {
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

func parseVolumeType(input string) (VolumeType, []string, error) {
	parts := strings.SplitN(input, "@", 2)
	vtype := VolumeType(parts[0])
	var mountOptions []string
	if len(parts) > 1 {
		mountOptions = strings.Split(parts[1], ",")
	}
	return vtype, mountOptions, maskAny(vtype.Validate())
}
