package jobs

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/juju/errgo"

	"arvika.pulcy.com/pulcy/deployit/units"
)

var (
	taskNamePattern = regexp.MustCompile(`^([a-z0-9_]{2,30})$`)
	secretsPath     = "/tmp/secrets"
	unitKindMain    = "-mn"
	unitKindSecrets = "-sc"
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

	Type             TaskType          `json:"type,omitempty" mapstructure:"type,omitempty"`
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

	return units, nil
}

// createMainUnit
func (t *Task) createMainUnit(ctx generatorContext) (*units.Unit, error) {
	name := t.containerName(ctx.ScalingGroup)
	image := t.Image.String()
	execStart, err := t.createMainDockerCmdLine(ctx)
	if err != nil {
		return nil, maskAny(err)
	}

	descriptionPostfix := fmt.Sprintf("[slice %d]", ctx.ScalingGroup)
	if t.group.Global {
		descriptionPostfix = "[global]"
	}
	main := &units.Unit{
		Name:         t.unitName(unitKindMain, strconv.Itoa(int(ctx.ScalingGroup))),
		FullName:     t.unitName(unitKindMain, strconv.Itoa(int(ctx.ScalingGroup))) + ".service",
		Description:  fmt.Sprintf("Main unit for %s %s", t.fullName(), descriptionPostfix),
		Type:         "service",
		Scalable:     t.group.IsScalable(),
		ScalingGroup: ctx.ScalingGroup,
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
		fmt.Sprintf("-/usr/bin/docker rm -f %s", t.containerName(ctx.ScalingGroup)),
	}
	for _, v := range t.Volumes {
		dir := strings.Split(v, ":")
		mkdir := fmt.Sprintf("/bin/sh -c 'test -e %s || mkdir -p %s'", dir[0], dir[0])
		main.ExecOptions.ExecStartPre = append(main.ExecOptions.ExecStartPre, mkdir)
	}

	main.ExecOptions.ExecStop = fmt.Sprintf("-/usr/bin/docker stop -t %v %s", main.ExecOptions.ContainerTimeoutStopSec, name)
	main.ExecOptions.ExecStopPost = []string{
		fmt.Sprintf("-/usr/bin/docker rm -f %s", name),
	}
	main.FleetOptions.IsGlobal = t.group.Global
	if t.group.IsScalable() && ctx.InstanceCount > 1 {
		main.FleetOptions.Conflicts(t.unitName(unitKindMain, "*") + ".service")
	}

	// Service dependencies
	// Requires=
	//main.ExecOptions.Require("flanneld.service")
	if requires, err := t.createMainRequires(ctx); err != nil {
		return nil, maskAny(err)
	} else {
		main.ExecOptions.Require(requires...)
	}
	main.ExecOptions.Require("docker.service")
	// After=...
	if after, err := t.createMainAfter(ctx); err != nil {
		return nil, maskAny(err)
	} else {
		main.ExecOptions.After(after...)
	}

	if err := t.addFrontEndRegistration(main); err != nil {
		return nil, maskAny(err)
	}

	return main, nil
}

// createMainDockerCmdLine creates the `ExecStart` line for
// the main unit.
func (t *Task) createMainDockerCmdLine(ctx generatorContext) ([]string, error) {
	serviceName := t.serviceName()
	image := t.Image.String()
	execStart := []string{
		"/usr/bin/docker",
		"run",
		"--rm",
		fmt.Sprintf("--name %s", t.containerName(ctx.ScalingGroup)),
	}
	if len(t.Ports) > 0 {
		for _, p := range t.Ports {
			execStart = append(execStart, fmt.Sprintf("-p %s", p))
		}
	} else {
		execStart = append(execStart, "-P")
	}
	for _, v := range t.Volumes {
		execStart = append(execStart, fmt.Sprintf("-v %s", v))
	}
	for _, secret := range t.Secrets {
		if ok, path := secret.TargetFile(); ok {
			hostPath, err := t.secretHostPath(secret, ctx.ScalingGroup)
			if err != nil {
				return nil, maskAny(err)
			}
			execStart = append(execStart, fmt.Sprintf("-v %s:%s:ro", hostPath, path))
		}
	}
	for _, name := range t.VolumesFrom {
		other, err := t.group.Task(name)
		if err != nil {
			return nil, maskAny(err)
		}
		execStart = append(execStart, fmt.Sprintf("--volumes-from %s", other.containerName(ctx.ScalingGroup)))
	}
	envArgs := []string{}
	for k, v := range t.Environment {
		envArgs = append(envArgs, "-e "+strconv.Quote(fmt.Sprintf("%s=%s", k, v)))
	}
	for _, secret := range t.Secrets {
		if ok, key := secret.TargetEnviroment(); ok {
			envArgs = append(envArgs, "-e "+strconv.Quote(fmt.Sprintf("%s=${%s}", key, key)))
		}
	}
	sort.Strings(envArgs)
	execStart = append(execStart, envArgs...)
	execStart = append(execStart, fmt.Sprintf("-e SERVICE_NAME=%s", serviceName)) // Support registrator
	for _, cap := range t.Capabilities {
		execStart = append(execStart, "--cap-add "+cap)
	}
	for _, ln := range t.Links {
		execStart = append(execStart, fmt.Sprintf("--add-host %s:${COREOS_PRIVATE_IPV4}", ln.PrivateDomainName()))
	}

	execStart = append(execStart, image)
	execStart = append(execStart, t.Args...)

	return execStart, nil
}

// createMainAfter creates the `After=` sequence for the main unit
func (t *Task) createMainAfter(ctx generatorContext) ([]string, error) {
	after := []string{
		"docker.service",
		"yard.service",
	}
	if len(t.Secrets) > 0 {
		secretsUnit := t.unitName(unitKindSecrets, strconv.Itoa(int(ctx.ScalingGroup))) + ".service"
		after = append(after, secretsUnit)
	}
	for _, name := range t.VolumesFrom {
		other, err := t.group.Task(name)
		if err != nil {
			return nil, maskAny(err)
		}
		after = append(after, other.unitName(unitKindMain, strconv.Itoa(int(ctx.ScalingGroup)))+".service")
	}

	return after, nil
}

// createMainRequires creates the `Requires=` sequence for the main unit
func (t *Task) createMainRequires(ctx generatorContext) ([]string, error) {
	requires := []string{
		"docker.service",
	}

	if len(t.Secrets) > 0 {
		secretsUnit := t.unitName(unitKindSecrets, strconv.Itoa(int(ctx.ScalingGroup))) + ".service"
		requires = append(requires, secretsUnit)
	}
	for _, name := range t.VolumesFrom {
		other, err := t.group.Task(name)
		if err != nil {
			return nil, maskAny(err)
		}
		requires = append(requires, other.unitName(unitKindMain, strconv.Itoa(int(ctx.ScalingGroup)))+".service")
	}

	return requires, nil
}

// createSecretsUnit creates a unit used to extract secrets from vault
func (t *Task) createSecretsUnit(ctx generatorContext) (*units.Unit, error) {
	descriptionPostfix := fmt.Sprintf("[slice %d]", ctx.ScalingGroup)
	if t.group.Global {
		descriptionPostfix = "[global]"
	}

	execStart := []string{"TODO"}
	unit := &units.Unit{
		Name:         t.unitName(unitKindSecrets, strconv.Itoa(int(ctx.ScalingGroup))),
		FullName:     t.unitName(unitKindSecrets, strconv.Itoa(int(ctx.ScalingGroup))) + ".service",
		Description:  fmt.Sprintf("Secrets unit for %s %s", t.fullName(), descriptionPostfix),
		Type:         "service",
		Scalable:     t.group.IsScalable(),
		ScalingGroup: ctx.ScalingGroup,
		ExecOptions:  units.NewExecOptions(execStart...),
		FleetOptions: units.NewFleetOptions(),
	}
	unit.ExecOptions.IsOneshot = true
	unit.ExecOptions.ExecStopPost = []string{
	// TODO cleanup
	//fmt.Sprintf("-/usr/bin/docker rm -f %s", name),
	}
	unit.FleetOptions.IsGlobal = t.group.Global
	if t.group.IsScalable() && ctx.InstanceCount > 1 {
		unit.FleetOptions.Conflicts(t.unitName(unitKindSecrets, "*") + ".service")
	}

	// Service dependencies
	// Requires=
	//main.ExecOptions.Require("flanneld.service")
	if requires, err := t.createMainRequires(ctx); err != nil {
		return nil, maskAny(err)
	} else {
		unit.ExecOptions.Require(requires...)
	}
	// After=...
	if after, err := t.createMainAfter(ctx); err != nil {
		return nil, maskAny(err)
	} else {
		unit.ExecOptions.After(after...)
	}

	return unit, nil
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
	Domain     string `json:"domain,omitempty"`
	PathPrefix string `json:"path-prefix,omitempty"`
	SslCert    string `json:"ssl-cert,omitempty"`
	Port       int    `json:"port,omitempty"`
	Private    bool   `json:"private,omitempty"`
}

// addFrontEndRegistration adds registration code for frontends to the given units
func (t *Task) addFrontEndRegistration(main *units.Unit) error {
	if len(t.PublicFrontEnds) == 0 && len(t.PrivateFrontEnds) == 0 {
		return nil
	}
	key := "/pulcy/frontend/" + t.serviceName()
	record := frontendRecord{
		Service:       t.serviceName(),
		HttpCheckPath: t.HttpCheckPath,
	}
	for _, fr := range t.PublicFrontEnds {
		record.Selectors = append(record.Selectors, frontendSelectorRecord{
			Domain:     fr.Domain,
			PathPrefix: fr.PathPrefix,
			SslCert:    fr.SslCert,
		})
	}
	for _, fr := range t.PrivateFrontEnds {
		record.Selectors = append(record.Selectors, frontendSelectorRecord{
			Domain:  t.privateDomainName(),
			Port:    fr.Port,
			Private: true,
		})
	}
	json, err := json.Marshal(&record)
	if err != nil {
		return maskAny(err)
	}
	main.ProjectSetting("FrontEndRegistration", key+"="+string(json))
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
