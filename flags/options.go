package flags

import (
	"fmt"
	"io/ioutil"
	"strings"

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
	return v, ok
}

func (o *Options) Set(raw string) error {
	if o.options == nil {
		o.options = make(map[string]string)
	}
	parts := strings.SplitN(raw, "=", 2)
	if len(parts) == 2 {
		// Normal key=value
		o.options[parts[0]] = parts[1]
		return nil
	}

	// Try option file
	_, err := ioutil.ReadFile(raw)
	if err != nil {
		return maskAny(errgo.WithCausef(nil, InvalidOptionError, raw))
	}

	// TODO PARSE
	return maskAny(errgo.WithCausef(nil, InvalidOptionError, "option files not yet supported"))

	//	return nil
}

func (o *Options) Type() string {
	return "options"
}
