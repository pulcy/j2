package jobs

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/juju/errgo"

	"arvika.pulcy.com/pulcy/deployit/units"
)

var (
	taskNamePattern = regexp.MustCompile(`^([a-z0-9_]{2,30})$`)
)

type TaskName string

func (tn TaskName) String() string {
	return string(tn)
}

func (tn TaskName) Validate() error {
	if !taskNamePattern.MatchString(string(tn)) {
		return maskAny(errgo.WithCausef(nil, InvalidNameError, "task name must match '%s', got '%s'", taskNamePattern, tn))
	}
	return nil
}

type Task struct {
	Name   TaskName   `json:"name", maspstructure:"-"`
	group  *TaskGroup `json:"-", mapstructure:"-"`
	Count  uint       `json:"-"` // This value is used during parsing only
	Global bool       `json:"-"` // This value is used during parsing only

	Image       DockerImage       `json:"image"`
	VolumesFrom []TaskName        `json:"volumes-from,omitempty"`
	Volumes     []string          `json:"volumes,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
}

type TaskList []*Task

// Check for errors
func (t *Task) Validate() error {
	if err := t.Name.Validate(); err != nil {
		return maskAny(err)
	}
	for _, name := range t.VolumesFrom {
		_, err := t.group.Task(name)
		if err != nil {
			return maskAny(err)
		}
	}

	return nil
}

// createUnits creates all units needed to run this task.
func (t *Task) createUnits(scalingGroup uint) ([]*units.Unit, error) {
	units := []*units.Unit{}
	main, err := t.createMainUnit(scalingGroup)
	if err != nil {
		return nil, maskAny(err)
	}
	units = append(units, main)

	return units, nil
}

// createMainUnit
func (t *Task) createMainUnit(scalingGroup uint) (*units.Unit, error) {
	execStart := []string{
		"/usr/bin/docker",
		"run",
		"--rm",
		"--name $NAME",
		"-P",
	}
	for _, v := range t.Volumes {
		execStart = append(execStart, fmt.Sprintf("-v %s", v))
	}
	for _, name := range t.VolumesFrom {
		other, err := t.group.Task(name)
		if err != nil {
			return nil, maskAny(err)
		}
		execStart = append(execStart, fmt.Sprintf("--volumes-from %s", other.containerName(scalingGroup)))
	}
	for k, v := range t.Environment {
		execStart = append(execStart, "-e "+strconv.Quote(fmt.Sprintf("%s=%s", k, v)))
	}
	execStart = append(execStart, "-e SERVICE_NAME=$NAME") // Support registrator
	execStart = append(execStart, "$IMAGE")
	execStart = append(execStart, t.Args...)
	main := &units.Unit{
		Name:         t.unitName(scalingGroup),
		FullName:     t.unitName(scalingGroup) + ".service",
		Description:  fmt.Sprintf("Main unit for %s slice %v", t.fullName(), scalingGroup),
		Type:         "service",
		Scalable:     t.group.IsScalable(),
		ScalingGroup: scalingGroup,
		ExecOptions:  units.NewExecOptions(execStart...),
		FleetOptions: units.NewFleetOptions(),
	}
	//main.FleetOptions.IsGlobal = ds.global
	main.ExecOptions.ExecStartPre = []string{
		"/usr/bin/docker pull $IMAGE",
		fmt.Sprintf("-/usr/bin/docker stop -t %v $NAME", main.ExecOptions.ContainerTimeoutStopSec),
		"-/usr/bin/docker rm -f $NAME",
	}
	main.ExecOptions.ExecStop = fmt.Sprintf("-/usr/bin/docker stop -t %v $NAME", main.ExecOptions.ContainerTimeoutStopSec)
	main.ExecOptions.ExecStopPost = []string{
		"-/usr/bin/docker rm -f $NAME",
	}
	main.ExecOptions.Environment = map[string]string{
		"NAME":  t.containerName(scalingGroup),
		"IMAGE": t.Image.String(),
	}
	main.FleetOptions.IsGlobal = t.group.Global

	return main, nil
}

// Gets the full name of this task: job/taskgroup/task
func (t *Task) fullName() string {
	return fmt.Sprintf("%s/%s", t.group.fullName(), t.Name)
}

// unitName returns the name of the systemd unit for this task.
func (t *Task) unitName(scalingGroup uint) string {
	base := strings.Replace(t.fullName(), "/", "-", -1)
	if !t.group.IsScalable() {
		return base
	}
	return fmt.Sprintf("%s@%v", base, scalingGroup)
}

// containerName returns the name of the docker contained used for this task.
func (t *Task) containerName(scalingGroup uint) string {
	base := strings.Replace(t.fullName(), "/", "-", -1)
	return fmt.Sprintf("%s-%v", base, scalingGroup)
}

func (l TaskList) Len() int {
	return len(l)
}

func (l TaskList) Less(i, j int) bool {
	return bytes.Compare([]byte(l[i].Name.String()), []byte(l[j].Name.String())) < 0
}

func (l TaskList) Swap(i, j int) {
	tmp := l[i]
	l[i] = l[j]
	l[j] = tmp
}
