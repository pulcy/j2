package jobs

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/juju/errgo"
)

type jobFunctions struct {
	jobPath string
}

// newJobFunctions creates a new instance of jobFunctions
func newJobFunctions(jobPath string) *jobFunctions {
	absJobPath, _ := filepath.Abs(jobPath)
	return &jobFunctions{
		jobPath: absJobPath,
	}
}

// Functions returns all supported template functions
func (jf *jobFunctions) Functions() template.FuncMap {
	return template.FuncMap{
		"cat":     jf.cat,
		"env":     jf.getEnv,
		"quote":   strconv.Quote,
		"replace": strings.Replace,
		"trim":    strings.TrimSpace,
	}
}

// getEnv loads an environment value and returns an error if it is empty.
func (jf *jobFunctions) getEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", errgo.WithCausef(nil, ValidationError, "Missing environment variables %s", key)
	}
	return value, nil
}

// cat returns the content of a file.
func (jf *jobFunctions) cat(path string) (string, error) {
	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(jf.jobDir(), path)
	}
	raw, err := ioutil.ReadFile(absPath)
	if os.IsNotExist(err) {
		return "", errgo.WithCausef(nil, ValidationError, "File '%s' not found", absPath)
	} else if err != nil {
		return "", maskAny(err)
	}
	return string(raw), nil
}

// jobDir returns the folder containing the current jobPath.
func (jf *jobFunctions) jobDir() string {
	return filepath.Dir(jf.jobPath)
}
