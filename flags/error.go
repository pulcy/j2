package flags

import (
	"github.com/juju/errgo"
)

var (
	InvalidOptionError = errgo.New("invalid option")
	ValidationError    = errgo.New("validation failed")

	maskAny = errgo.MaskFunc(errgo.Any)
)
