package jobs

import (
	"encoding/json"
	"io/ioutil"
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
	Name   JobName                      `json:"name"`
	Groups map[TaskGroupName]*TaskGroup `json:"groups"`
}

// ParseJob takes input from a given reader and parses it into a Job.
func ParseJob(input []byte) (Job, error) {
	job := Job{}
	if err := json.Unmarshal(input, &job); err != nil {
		return Job{}, maskAny(err)
	}
	return job, nil
}

// ParseJobFromFile reads a job from file
func ParseJobFromFile(path string) (Job, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return Job{}, maskAny(err)
	}
	job, err := ParseJob(data)
	if err != nil {
		return Job{}, maskAny(err)
	}
	return job, nil
}

// Link objects just after parsing
func (j *Job) link() {
	for k, v := range j.Groups {
		v.Name = k
		v.Job = j
		v.link()
	}
}

// Check for errors
func (j *Job) Validate() error {
	if err := j.Name.Validate(); err != nil {
		return maskAny(err)
	}
	for _, tg := range j.Groups {
		err := tg.Validate()
		if err != nil {
			return maskAny(err)
		}
	}
	return nil
}
