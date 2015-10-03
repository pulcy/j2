package jobs

import (
	"regexp"

	"github.com/juju/errgo"
)

var (
	jobNamePattern = regexp.MustCompile(`^([a-z0-9_]{3,30})$`)
)

type JobName string

func (jn JobName) String() string {
	return string(jn)
}

func (jn JobName) Validate() error {
	if !jobNamePattern.MatchString(string(jn)) {
		return maskAny(errgo.WithCausef(nil, InvalidNameError, "job name must match '%s', got '%s'", jobNamePattern, jn))
	}
	return nil
}

type Job struct {
	Name   JobName
	Groups []*TaskGroup
}

// Link objects just after parsing
func (j *Job) link() {
	for _, tg := range j.Groups {
		tg.job = j
		tg.link()
	}
}

// Check for errors
func (j *Job) Validate() error {
	if err := j.Name.Validate(); err != nil {
		return maskAny(err)
	}
	if len(j.Groups) == 0 {
		return maskAny(errgo.WithCausef(nil, ValidationError, "job has no groups"))
	}
	for _, tg := range j.Groups {
		err := tg.Validate()
		if err != nil {
			return maskAny(err)
		}
	}
	return nil
}

func (j *Job) Generate(groups []TaskGroupName, currentScalingGroup uint) *Generator {
	return newGenerator(j, groups, currentScalingGroup)
}

func (j *Job) MaxCount() uint {
	count := uint(0)
	for _, tg := range j.Groups {
		if tg.Count > count {
			count = tg.Count
		}
	}
	return count
}
