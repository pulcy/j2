package jobs

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/juju/errgo"

	"arvika.pulcy.com/pulcy/deployit/units"
)

var (
	taskNamePattern = regexp.MustCompile(`^([a-z0-9_]{3,30})$`)
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
	Name  TaskName   `json:"-"`
	Group *TaskGroup `json:"-"`

	Image       DockerImage       `json:"image"`
	VolumesFrom []TaskName        `json:"volumes-from"`
	Ports       []Port            `json:"ports"`
	Args        []string          `json:"args"`
	Environment map[string]string `json:"environment"`
}

// Check for errors
func (t *Task) Validate() error {
	if err := t.Name.Validate(); err != nil {
		return maskAny(err)
	}
	for _, name := range t.VolumesFrom {
		_, err := t.Group.Task(name)
		if err != nil {
			return maskAny(err)
		}
	}

	return nil
}

// createUnits
func (t *Task) createUnits(scalingGroup uint8) ([]units.Unit, error) {
	units := []units.Unit{}
	main, err := t.createMainUnit(scalingGroup)
	if err != nil {
		return nil, maskAny(err)
	}
	units = append(units, main)

	return units, nil
}

// createMainUnit
func (t *Task) createMainUnit(scalingGroup uint8) (units.Unit, error) {
	execStart := []string{
		"/usr/bin/docker",
		"run",
		"--rm",
		"--name $NAME",
	}
	for _, p := range t.Ports {
		execStart = append(execStart, fmt.Sprintf("-p ${COREOS_PRIVATE_IPV4}::%s", p.String()))
	}
	/*for _, v := range ds.volumes {
		execStart = append(execStart, fmt.Sprintf("-v %s:%s", v.hostPath, v.containerPath))
	}*/
	for _, name := range t.VolumesFrom {
		other, err := t.Group.Task(name)
		if err != nil {
			return units.Unit{}, maskAny(err)
		}
		execStart = append(execStart, fmt.Sprintf("--volumes-from %s", other.containerName(scalingGroup)))
	}
	for k, v := range t.Environment {
		execStart = append(execStart, "-e "+strconv.Quote(fmt.Sprintf("%s=%s", k, v)))
	}
	execStart = append(execStart, "-e SERVICE_NAME=$NAME") // Support registrator
	execStart = append(execStart, "$IMAGE")
	execStart = append(execStart, t.Args...)
	main := units.Unit{
		Name:         t.unitName(scalingGroup),
		Description:  fmt.Sprintf("Main unit for %s/%s/%s", t.fullName()),
		Type:         "service",
		Scalable:     t.Group.Count > 1,
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

	return main, nil
}

// Gets the full name of this task: job/taskgroup/task
func (t *Task) fullName() string {
	return fmt.Sprintf("%s/%s", t.Group.fullName(), t.Name)
}

// unitName returns the name of the systemd unit for this task.
func (t *Task) unitName(scalingGroup uint8) string {
	return fmt.Sprintf("%s@%v.service", t.Name.String(), scalingGroup)
}

// containerName returns the name of the docker contained used for this task.
func (t *Task) containerName(scalingGroup uint8) string {
	return fmt.Sprintf("%s@%v", t.Name.String(), scalingGroup)
}
