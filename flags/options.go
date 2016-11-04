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

package flags

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/hashicorp/hcl"
	"github.com/juju/errgo"
)

type Options struct {
	options map[string]interface{}
}

func (o *Options) String() string {
	lines := []string{}
	for k, v := range o.options {
		lines = append(lines, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(lines, ", ")
}

func (o *Options) Get(key string) (interface{}, bool) {
	v, ok := o.options[key]
	if ok {
		return v, ok
	}
	envKey := strings.ToUpper(strings.Replace(key, "-", "_", -1))
	value := os.Getenv(envKey)
	if value != "" {
		return value, true
	}
	return "", false
}

func (o *Options) Set(raw string) error {
	if strings.Contains(raw, "=") {
		// Normal key=value
		if err := o.parseKeyValue(raw); err != nil {
			return maskAny(err)
		}
		return nil
	}

	// Try option file
	err := o.parseFile(raw)
	if err != nil {
		return maskAny(errgo.WithCausef(err, InvalidOptionError, raw))
	}

	return nil
}

func (o *Options) SetKeyValue(key string, value interface{}) {
	if o.options == nil {
		o.options = make(map[string]interface{})
	}
	o.options[key] = value
}

func (o *Options) Type() string {
	return "options"
}

func (o *Options) parseKeyValue(raw string) error {
	if err := o.parse(raw); err == nil {
		return nil
	}
	// Fallback to simple K=V
	parts := strings.SplitN(raw, "=", 2)
	if len(parts) == 2 {
		o.SetKeyValue(parts[0], parts[1])
		return nil
	}
	return maskAny(fmt.Errorf("Unknown input '%s", raw))
}

func (o *Options) parseFile(path string) error {
	// Read the file
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return maskAny(err)
	}

	// Parse the input
	if err := o.parse(string(data)); err != nil {
		return maskAny(err)
	}

	return nil
}

func (o *Options) parse(content string) error {
	// Parse the input
	obj, err := hcl.Parse(content)
	if err != nil {
		return maskAny(err)
	}

	// Parse hcl into options
	// Decode the full thing into a map[string]interface for ease
	var m map[string]interface{}
	if err := hcl.DecodeObject(&m, obj); err != nil {
		return maskAny(err)
	}

	// Merge key/value pairs into myself
	for k, v := range m {
		if mapArray, ok := v.([]map[string]interface{}); ok && len(mapArray) == 1 {
			v = mapArray[0]
		}
		o.SetKeyValue(k, v)
	}

	return nil
}
