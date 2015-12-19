package jobs

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/juju/errgo"

	"arvika.pulcy.com/pulcy/deployit/units"
)

const (
	secretsPath     = "/tmp/secrets"
	unitKindMain    = "-mn"
	unitKindSecrets = "-sc"
	unitKindTimer   = "-ti"
)

var (
	commonAfter = []string{
		"docker.service",
		"yard.service",
	}
	commonRequires = []string{
		"docker.service",
	}
)

type Task struct {
	Name   TaskName   `json:"name", maspstructure:"-"`
	group  *TaskGroup `json:"-", mapstructure:"-"`
	Count  uint       `json:"-"` // This value is used during parsing only
	Global bool       `json:"-"` // This value is used during parsing only

	Type             TaskType          `json:"type,omitempty" mapstructure:"type,omitempty"`
	Timer            string            `json:"timer,omitempty" mapstructure:"timer,omitempty"`
	Image            DockerImage       `json:"image"`
	VolumesFrom      []TaskName        `json:"volumes-from,omitempty"`
	Volumes          []string          `json:"volumes,omitempty"`
	Args             []string          `json:"args,omitempty"`
	Environment      map[string]string `json:"environment,omitempty"`
	Ports            []string          `json:"ports,omitempty"`
	PublicFrontEnds  []PublicFrontEnd  `json:"frontends,omitempty"`
	PrivateFrontEnds []PrivateFrontEnd `json:"private-frontends,omitempty"`
	HttpCheckPath    string            `json:"http-check-path,omitempty" mapstructure:"http-check-path,omitempty"`
	Capabilities     []string          `json:"capabilities,omitempty"`
	Links            []LinkName        `json:"links,omitempty"`
	Secrets          []Secret          `json:"secrets,omitempty"`
}

// Check for errors
func (t *Task) Validate() error {
	if err := t.Name.Validate(); err != nil {
		return maskAny(err)
	}
	if err := t.Type.Validate(); err != nil {
		return maskAny(err)
	}
	for _, name := range t.VolumesFrom {
		_, err := t.group.Task(name)
		if err != nil {
			return maskAny(err)
		}
	}
	for _, ln := range t.Links {
		if err := ln.Validate(); err != nil {
			return maskAny(err)
		}
	}
	for _, f := range t.PublicFrontEnds {
		if err := f.Validate(); err != nil {
			return maskAny(err)
		}
	}
	for _, f := range t.PrivateFrontEnds {
		if err := f.Validate(); err != nil {
			return maskAny(err)
		}
	}
	for _, s := range t.Secrets {
		if err := s.Validate(); err != nil {
			return maskAny(err)
		}
	}
	if t.Timer != "" {
		if t.Type != "oneshot" {
			return maskAny(errgo.WithCausef(nil, ValidationError, "timer only valid in combination with oneshot (in '%s')", t.Name))
		}
	}
	return nil
}

// createUnits creates all units needed to run this task.
func (t *Task) createUnits(ctx generatorContext) ([]*units.Unit, error) {
	units := []*units.Unit{}

	if len(t.Secrets) > 0 {
		unit, err := t.createSecretsUnit(ctx)
		if err != nil {
			return nil, maskAny(err)
		}
		units = append(units, unit)
	}

	main, err := t.createMainUnit(ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	units = append(units, main)

	timer, err := t.createTimerUnit(ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	if timer != nil {
		units = append(units, timer)
	}

	return units, nil
}

// Gets the full name of this task: job/taskgroup/task
func (t *Task) fullName() string {
	return fmt.Sprintf("%s/%s", t.group.fullName(), t.Name)
}

// privateDomainName returns the DNS name (in the private namespace) for the given task.
func (t *Task) privateDomainName() string {
	ln := NewLinkName(t.group.job.Name, t.group.Name, t.Name)
	return ln.PrivateDomainName()
}

// secretHostPath creates the path of the host of a given secret for the given task.
func (t *Task) secretHostPath(secret Secret, scalingGroup uint) (string, error) {
	hash, err := secret.hash()
	if err != nil {
		return "", maskAny(err)
	}
	return filepath.Join(secretsPath, t.containerName(scalingGroup), hash), nil
}

// unitName returns the name of the systemd unit for this task.
func (t *Task) unitName(kind string, scalingGroup string) string {
	base := strings.Replace(t.fullName(), "/", "-", -1) + kind
	if !t.group.IsScalable() {
		return base
	}
	return fmt.Sprintf("%s@%s", base, scalingGroup)
}

// unitDescription creates the description of a unit
func (t *Task) unitDescription(prefix string, scalingGroup uint) string {
	descriptionPostfix := fmt.Sprintf("[slice %d]", scalingGroup)
	if t.group.Global {
		descriptionPostfix = "[global]"
	}
	return fmt.Sprintf("%s unit for %s %s", prefix, t.fullName(), descriptionPostfix)
}

// containerName returns the name of the docker contained used for this task.
func (t *Task) containerName(scalingGroup uint) string {
	base := strings.Replace(t.fullName(), "/", "-", -1)
	if !t.group.IsScalable() {
		return base
	}
	return fmt.Sprintf("%s-%v", base, scalingGroup)
}

// serviceName returns the name used to register this service.
func (t *Task) serviceName() string {
	return strings.Replace(t.fullName(), "/", "-", -1)
}
