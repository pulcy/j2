package jobs

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/dchest/uniuri"
	"github.com/nyarla/go-crypt"

	"arvika.pulcy.com/pulcy/deployit/units"
)

// createMainUnit
func (t *Task) createMainUnit(ctx generatorContext) (*units.Unit, error) {
	name := t.containerName(ctx.ScalingGroup)
	image := t.Image.String()
	execStart, err := t.createMainDockerCmdLine(ctx)
	if err != nil {
		return nil, maskAny(err)
	}

	main := &units.Unit{
		Name:         t.unitName(unitKindMain, strconv.Itoa(int(ctx.ScalingGroup))),
		FullName:     t.unitName(unitKindMain, strconv.Itoa(int(ctx.ScalingGroup))) + ".service",
		Description:  t.unitDescription("Main", ctx.ScalingGroup),
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
	after := append([]string{}, commonAfter...)

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
	requires := append([]string{}, commonRequires...)

	for _, name := range t.VolumesFrom {
		other, err := t.group.Task(name)
		if err != nil {
			return nil, maskAny(err)
		}
		requires = append(requires, other.unitName(unitKindMain, strconv.Itoa(int(ctx.ScalingGroup)))+".service")
	}

	return requires, nil
}

type frontendRecord struct {
	Selectors     []frontendSelectorRecord `json:"selectors"`
	Service       string                   `json:"service,omitempty"`
	HttpCheckPath string                   `json:"http-check-path,omitempty"`
}

type frontendSelectorRecord struct {
	Domain     string       `json:"domain,omitempty"`
	PathPrefix string       `json:"path-prefix,omitempty"`
	SslCert    string       `json:"ssl-cert,omitempty"`
	Port       int          `json:"port,omitempty"`
	Private    bool         `json:"private,omitempty"`
	Users      []userRecord `json:"users,omitempty"`
}

type userRecord struct {
	Name         string `json:"user"`
	PasswordHash string `json:"pwhash"`
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
		selRecord := frontendSelectorRecord{
			Domain:     fr.Domain,
			PathPrefix: fr.PathPrefix,
			SslCert:    fr.SslCert,
			Port:       fr.Port,
		}
		selRecord.addUsers(fr.Users)
		record.Selectors = append(record.Selectors, selRecord)
	}
	for _, fr := range t.PrivateFrontEnds {
		selRecord := frontendSelectorRecord{
			Domain:  t.privateDomainName(),
			Port:    fr.Port,
			Private: true,
		}
		selRecord.addUsers(fr.Users)
		record.Selectors = append(record.Selectors, selRecord)
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

// addUsers adds the given users to the selector record, while encrypting the passwords.
func (selRecord *frontendSelectorRecord) addUsers(users []User) {
	for _, u := range users {
		salt := uniuri.New()
		userRec := userRecord{
			Name:         u.Name,
			PasswordHash: crypt.Crypt(u.Password, salt),
		}
		selRecord.Users = append(selRecord.Users, userRec)
	}
}
