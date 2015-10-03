package jobs

import (
	"github.com/juju/errgo"
)

var (
	TaskNotFoundError = errgo.New("task not found")
	InvalidNameError  = errgo.New("invalid name")
	ValidationError   = errgo.New("validation failed")

	maskAny = errgo.MaskFunc(errgo.Any)
)
