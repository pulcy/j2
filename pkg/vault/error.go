package vault

import (
	"github.com/juju/errgo"
)

var (
	InvalidArgumentError = errgo.New("invalid argument error")
	VaultError           = errgo.New("vault error")
	maskAny              = errgo.MaskFunc(errgo.Any)
)
