package jobs

import (
	"os"
	"text/template"

	"github.com/juju/errgo"
)

// First we create a FuncMap with which to register the function.
var (
	funcMap = template.FuncMap{
		"env": getEnv,
	}
)

func getEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", errgo.WithCausef(nil, ValidationError, "Missing environment variables %s", key)
	}
	return value, nil
}
