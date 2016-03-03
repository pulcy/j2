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

	"encoding/json"
	"regexp"
	"strings"
)

var (
	// https://github.com/docker/docker/blob/6d6dc2c1a1e36a2c3992186664d4513ce28e84ae/registry/registry.go#L27
	PatternNamespace = regexp.MustCompile(`^([a-z0-9_]{4,30})$`)
	PatternImage     = regexp.MustCompile(`^([a-z0-9-_.]+)$`)
	PatternVersion   = regexp.MustCompile("^[a-zA-Z0-9-\\._]+$")
)

var (
	// NOTE: This format description is slightly different from what we parse down there.
	// The format given here is the one docker documents. But the repository also consist of a
	// namespace which is more or less always there. Since our business logic requires some checks based on the
	// namespace, we parse it explicitly.
	ErrInvalidFormat = errgo.New("Not a valid docker image. Format: [<registry>/]<repository>[:<version>]")
)

func MustParseDockerImage(image string) DockerImage {
	img, err := ParseDockerImage(image)
	if err != nil {
		panic(errgo.Mask(err))
	}
	return img
}

func ParseDockerImage(image string) (DockerImage, error) {
	var dockerImage DockerImage
	err := dockerImage.parse(image)
	return dockerImage, err
}

type DockerImage struct {
	Registry   string // The registry name
	Namespace  string // The namespace
	Repository string // The repository name
	Version    string // The version part
}

func (i DockerImage) replaceVariables(ctx *variableContext) DockerImage {
	i.Registry = ctx.replaceString(i.Registry)
	i.Namespace = ctx.replaceString(i.Namespace)
	i.Repository = ctx.replaceString(i.Repository)
	i.Version = ctx.replaceString(i.Version)
	return i
}

func (img DockerImage) MarshalJSON() ([]byte, error) {
	return json.Marshal(img.String())
}

func (img *DockerImage) UnmarshalJSON(data []byte) error {
	var input string

	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}
	return img.parse(input)
}

func (img DockerImage) DefaultLatestVersion() DockerImage {
	if img.Version == "" {
		img.Version = "latest"
	}
	return img
}

// Validate checks that the given docker image is valid.
// Returns nil if valid, or an error if not valid.
func (img DockerImage) Validate() error {
	tmp := DockerImage{}
	return tmp.parse(img.String())
}

func (img *DockerImage) parse(input string) error {
	if len(input) == 0 {
		return errgo.Notef(ErrInvalidFormat, "Zero length")
	}
	if strings.Contains(input, " ") {
		return errgo.Notef(ErrInvalidFormat, "No whitespaces allowed")
	}

	splitByPath := strings.Split(input, "/")
	if len(splitByPath) > 3 {
		return errgo.Notef(ErrInvalidFormat, "Too many path elements")
	}

	if containsRegistry(splitByPath) {
		img.Registry = splitByPath[0]
		splitByPath = splitByPath[1:]
	}

	switch len(splitByPath) {
	case 1:
		img.Repository = splitByPath[0]
	case 2:
		img.Namespace = splitByPath[0]
		img.Repository = splitByPath[1]

		if !isNamespace(img.Namespace) {
			return errgo.Notef(ErrInvalidFormat, "Invalid namespace part: "+img.Namespace)
		}
	default:
		return errgo.Notef(ErrInvalidFormat, "Invalid format")
	}

	// Now split img.Repository into img.Repository and img.Version
	splitByVersionSeparator := strings.Split(img.Repository, ":")

	switch len(splitByVersionSeparator) {
	case 2:
		img.Repository = splitByVersionSeparator[0]
		img.Version = splitByVersionSeparator[1]

		if !isVersion(img.Version) {
			return errgo.Notef(ErrInvalidFormat, "Invalid version %#v", img.Version)
		}
	case 1:
		img.Repository = splitByVersionSeparator[0]
		// Don't apply latest, since we would produce an extra diff, when checking
		// for arbitrary keys in the app-config. 'latest' is not necessary anyway,
		// because the docker daemon does not pull all tags of an image, if there
		// is none given.
		img.Version = ""
	default:
		return errgo.Notef(ErrInvalidFormat, "Too many double colons")
	}

	if !isImage(img.Repository) {
		return errgo.Notef(ErrInvalidFormat, "Invalid image part %#v", img.Repository)
	}
	return nil
}

// Returns all image inforamtion combined: <registry>/<namespace>/<repository>:<version>
func (img DockerImage) String() string {
	var imageString string

	if img.Registry != "" {
		imageString += img.Registry + "/"
	}

	if img.Namespace != "" {
		imageString += img.Namespace + "/"
	}

	imageString += img.Repository

	if img.Version != "" {
		imageString += ":" + img.Version
	}

	return imageString
}

func containsRegistry(input []string) bool {
	// See https://github.com/docker/docker/blob/6d6dc2c1a1e36a2c3992186664d4513ce28e84ae/registry/registry.go#L204
	if len(input) == 1 {
		return false
	}
	if len(input) == 2 && strings.ContainsAny(input[0], ".:") {
		return true
	}
	if len(input) == 3 {
		return true
	}
	return false
}

func isNamespace(input string) bool {
	if len(input) < 4 {
		return false
	}
	return PatternNamespace.MatchString(input)
}

func isImage(input string) bool {
	if len(input) == 0 {
		return false
	}
	return PatternImage.MatchString(input)
}

func isVersion(input string) bool {
	return PatternVersion.MatchString(input)
}
