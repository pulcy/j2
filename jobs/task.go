package jobs

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
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

type TaskType string

func (tt TaskType) String() string {
	return string(tt)
}

func (tt TaskType) Validate() error {
	switch tt {
	case "", "oneshot":
		return nil
	default:
		return maskAny(errgo.WithCausef(nil, ValidationError, "type has invalid value '%s'", tt))
	}
}

type Task struct {
	Name   TaskName   `json:"name", maspstructure:"-"`
	group  *TaskGroup `json:"-", mapstructure:"-"`
	Count  uint       `json:"-"` // This value is used during parsing only
	Global bool       `json:"-"` // This value is used during parsing only

	Type          TaskType          `json:"type,omitempty" mapstructure:"type,omitempty"`
	Image         DockerImage       `json:"image"`
	VolumesFrom   []TaskName        `json:"volumes-from,omitempty"`
	Volumes       []string          `json:"volumes,omitempty"`
	Args          []string          `json:"args,omitempty"`
	Environment   map[string]string `json:"environment,omitempty"`
	Ports         []string          `json:"ports,omitempty"`
	FrontEnds     []FrontEnd        `json:"frontends,omitempty"`
	HttpCheckPath string            `json:"http-check-path,omitempty" mapstructure:"http-check-path,omitempty"`
}

type TaskList []*Task

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
	for _, f := range t.FrontEnds {
		if err := f.Validate(); err != nil {
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
	name := t.containerName(scalingGroup)
	serviceName := t.serviceName()
	image := t.Image.String()
	execStart := []string{
		"/usr/bin/docker",
		"run",
		"--rm",
		fmt.Sprintf("--name %s", name),
	}
	if len(t.Ports) > 0 {
		for _, p := range t.Ports {
			execStart = append(execStart, fmt.Sprintf("-p %s", p))
		}
	} else {
		execStart = append(execStart, "-P")
	}
	after := []string{}
	for _, v := range t.Volumes {
		execStart = append(execStart, fmt.Sprintf("-v %s", v))
	}
	for _, name := range t.VolumesFrom {
		other, err := t.group.Task(name)
		if err != nil {
			return nil, maskAny(err)
		}
		execStart = append(execStart, fmt.Sprintf("--volumes-from %s", other.containerName(scalingGroup)))
		after = append(after, other.unitName(strconv.Itoa(int(scalingGroup)))+".service")
	}
	for k, v := range t.Environment {
		execStart = append(execStart, "-e "+strconv.Quote(fmt.Sprintf("%s=%s", k, v)))
	}
	execStart = append(execStart, fmt.Sprintf("-e SERVICE_NAME=%s", serviceName)) // Support registrator
	execStart = append(execStart, image)
	execStart = append(execStart, t.Args...)
	main := &units.Unit{
		Name:         t.unitName(strconv.Itoa(int(scalingGroup))),
		FullName:     t.unitName(strconv.Itoa(int(scalingGroup))) + ".service",
		Description:  fmt.Sprintf("Main unit for %s slice %v", t.fullName(), scalingGroup),
		Type:         "service",
		Scalable:     t.group.IsScalable(),
		ScalingGroup: scalingGroup,
		ExecOptions:  units.NewExecOptions(execStart...),
		FleetOptions: units.NewFleetOptions(),
	}
	switch t.Type {
	case "oneshot":
		main.ExecOptions.IsOneshot = true
	}
	//main.FleetOptions.IsGlobal = ds.global
	main.ExecOptions.ExecStartPre = []string{
		fmt.Sprintf("/usr/bin/docker pull %s", image),
		fmt.Sprintf("-/usr/bin/docker stop -t %v %s", main.ExecOptions.ContainerTimeoutStopSec, name),
		fmt.Sprintf("-/usr/bin/docker rm -f %s", name),
	}
	for _, v := range t.Volumes {
		dir := strings.Split(v, ":")
		mkdir := fmt.Sprintf("/bin/sh -c 'test -e %s || mkdir -p %s'", dir[0], dir[0])
		main.ExecOptions.ExecStartPre = append(main.ExecOptions.ExecStartPre, mkdir)
	}
	for _, name := range t.VolumesFrom {
		other, err := t.group.Task(name)
		if err != nil {
			return nil, maskAny(err)
		}
		main.ExecOptions.Require(other.containerName(scalingGroup))
	}
	main.ExecOptions.ExecStop = fmt.Sprintf("-/usr/bin/docker stop -t %v %s", main.ExecOptions.ContainerTimeoutStopSec, name)
	main.ExecOptions.ExecStopPost = []string{
		fmt.Sprintf("-/usr/bin/docker rm -f %s", name),
	}
	main.FleetOptions.IsGlobal = t.group.Global
	if t.group.IsScalable() {
		main.FleetOptions.Conflicts(t.unitName("*") + ".service")
	}
	//main.ExecOptions.Require("flanneld.service")
	main.ExecOptions.Require("docker.service")
	main.ExecOptions.After("docker.service")
	main.ExecOptions.After("yard.service")
	main.ExecOptions.After(after...)

	if err := t.addFrontEndRegistration(main); err != nil {
		return nil, maskAny(err)
	}

	return main, nil
}

// Gets the full name of this task: job/taskgroup/task
func (t *Task) fullName() string {
	return fmt.Sprintf("%s/%s", t.group.fullName(), t.Name)
}

// unitName returns the name of the systemd unit for this task.
func (t *Task) unitName(scalingGroup string) string {
	base := strings.Replace(t.fullName(), "/", "-", -1)
	if !t.group.IsScalable() {
		return base
	}
	return fmt.Sprintf("%s@%s", base, scalingGroup)
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

type frontendRecord struct {
	Selectors     []frontendSelectorRecord `json:"selectors"`
	Service       string                   `json:"service,omitempty"`
	HttpCheckPath string                   `json:"http-check-path,omitempty"`
}

type frontendSelectorRecord struct {
	Domain      string `json:"domain,omitempty"`
	PathPrefix  string `json:"path-prefix,omitempty"`
	SslCert     string `json:"ssl-cert,omitempty"`
	PrivatePort int    `json:"private-port,omitempty"`
}

// addFrontEndRegistration adds registration code for frontends to the given units
func (t *Task) addFrontEndRegistration(main *units.Unit) error {
	if len(t.FrontEnds) == 0 {
		return nil
	}
	key := "/pulcy/frontend/" + t.serviceName()
	record := frontendRecord{
		Service:       t.serviceName(),
		HttpCheckPath: t.HttpCheckPath,
	}
	for _, fr := range t.FrontEnds {
		record.Selectors = append(record.Selectors, frontendSelectorRecord{
			Domain:      fr.Domain,
			PathPrefix:  fr.PathPrefix,
			SslCert:     fr.SslCert,
			PrivatePort: fr.PrivatePort,
		})
	}
	json, err := json.Marshal(&record)
	if err != nil {
		return maskAny(err)
	}
	main.ExecOptions.ExecStartPost = append(main.ExecOptions.ExecStartPost,
		fmt.Sprintf("/bin/sh -c 'echo %s | base64 -d | /usr/bin/etcdctl set %s'", base64.StdEncoding.EncodeToString(json), key),
	)
	return nil
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
