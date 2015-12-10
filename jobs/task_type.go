package jobs

import (
	"github.com/juju/errgo"
)

type TaskType string

func (tt TaskType) String() string {
	return string(tt)
}

func (tt TaskType) Validate() error {
	switch tt {
	case "", "oneshot":
		return nil
	default:
		return maskAny(errgo.WithCausef(nil, ValidationError, "type has invalid value '%s'", tt))
	}
}
