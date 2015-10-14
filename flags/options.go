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
	options map[string]string
}

func (o *Options) String() string {
	lines := []string{}
	for k, v := range o.options {
		lines = append(lines, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(lines, ", ")
}

func (o *Options) Get(key string) (string, bool) {
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
	parts := strings.SplitN(raw, "=", 2)
	if len(parts) == 2 {
		// Normal key=value
		o.set(parts[0], parts[1])
		return nil
	}

	// Try option file
	err := o.parseFile(raw)
	if err != nil {
		return maskAny(errgo.WithCausef(err, InvalidOptionError, raw))
	}

	return nil
}

func (o *Options) set(key, value string) {
	if o.options == nil {
		o.options = make(map[string]string)
	}
	// Normal key=value
	o.options[key] = value
}

func (o *Options) Type() string {
	return "options"
}

func (o *Options) parseFile(path string) error {
	// Read the file
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return maskAny(err)
	}

	// Parse the input
	obj, err := hcl.Parse(string(data))
	if err != nil {
		return maskAny(err)
	}

	// Parse hcl into options
	// Decode the full thing into a map[string]interface for ease
	var m map[string]string
	if err := hcl.DecodeObject(&m, obj); err != nil {
		return maskAny(err)
	}

	// Merge key/value pairs into myself
	for k, v := range m {
		o.set(k, v)
	}

	return nil
}
