package flags

import (
	"github.com/juju/errgo"
)

var (
	InvalidOptionError = errgo.New("invalid option")

	maskAny = errgo.MaskFunc(errgo.Any)
)
