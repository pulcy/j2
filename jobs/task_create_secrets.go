package jobs

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"arvika.pulcy.com/pulcy/deployit/units"
)

// createSecretsUnit creates a unit used to extract secrets from vault
func (t *Task) createSecretsUnit(ctx generatorContext) (*units.Unit, error) {
	// Create all secret extraction commands
	jobID := t.group.job.ID
	if jobID == "" {
		return nil, maskAny(fmt.Errorf("job ID missing for job %s with secrets", t.group.job.Name))
	}
	cmds := [][]string{}
	envPaths := []string{}
	for _, secret := range t.Secrets {
		if ok, _ := secret.TargetFile(); ok {
			targetPath, err := t.secretFilePath(ctx.ScalingGroup, secret)
			if err != nil {
				return nil, maskAny(err)
			}
			cmd := []string{
				"/usr/bin/docker",
				"run",
				"--rm",
				"-v", "/etc/pulcy/cluster-id:/etc/pulcy/cluster-id:ro",
				"-v", "/etc/machine-id:/etc/machine-id:ro",
				ctx.Images.VaultMonkey,
				"extract",
				"file",
				"--target", targetPath,
				"--job-id", jobID,
				secret.VaultPath(),
			}
			cmds = append(cmds, cmd)
		} else if ok, environmentKey := secret.TargetEnviroment(); ok {
			envPaths = append(envPaths, fmt.Sprintf("%s=%s", environmentKey, secret.VaultPath()))
		}
	}
	if len(envPaths) > 0 {
		cmd := append([]string{
			"/usr/bin/docker",
			"run",
			"--rm",
			"-v", "/etc/pulcy/cluster-id:/etc/pulcy/cluster-id:ro",
			"-v", "/etc/machine-id:/etc/machine-id:ro",
			ctx.Images.VaultMonkey,
			"extract",
			"env",
			"--job-id", jobID,
		}, envPaths...)
		cmds = append(cmds, cmd)
	}

	// Use last comand as ExecStart
	execStart := cmds[len(cmds)-1]
	unit := &units.Unit{
		Name:         t.unitName(unitKindSecrets, strconv.Itoa(int(ctx.ScalingGroup))),
		FullName:     t.unitName(unitKindSecrets, strconv.Itoa(int(ctx.ScalingGroup))) + ".service",
		Description:  t.unitDescription("Secrets", ctx.ScalingGroup),
		Type:         "service",
		Scalable:     t.group.IsScalable(),
		ScalingGroup: ctx.ScalingGroup,
		ExecOptions:  units.NewExecOptions(execStart...),
		FleetOptions: units.NewFleetOptions(),
	}
	unit.ExecOptions.IsOneshot = true
	if len(cmds) > 1 {
		// Use all but last as ExecStartPre commands
		for _, cmd := range cmds[:len(cmds)-1] {
			unit.ExecOptions.ExecStartPre = append(unit.ExecOptions.ExecStartPre, strings.Join(cmd, " "))
		}
	}
	unit.FleetOptions.IsGlobal = t.group.Global

	// Service dependencies
	// Requires=
	unit.ExecOptions.Require(commonRequires...)
	// After=...
	unit.ExecOptions.After(commonAfter...)

	return unit, nil
}

// secretsRootPath returns the path of the root directory that will contain secret files for the given task.
func (t *Task) secretsRootPath(scalingGroup uint) string {
	return filepath.Join("/tmp/secrets", t.containerName(scalingGroup))
}

// secretEnvironmentPath returns the path of the file containing all secret environment variables
// for the given container.
func (t *Task) secretEnvironmentPath(scalingGroup uint) string {
	return filepath.Join(t.secretsRootPath(scalingGroup), "environment")
}

// secretFilePath returns the path of the file containing the given secret (file type)
func (t *Task) secretFilePath(scalingGroup uint, secret Secret) (string, error) {
	if secret.File == "" {
		return "", maskAny(fmt.Errorf("Wrong secret, file must be non-empty"))
	}
	hash, err := secret.hash()
	if err != nil {
		return "", maskAny(err)
	}
	return filepath.Join(t.secretsRootPath(scalingGroup), hash), nil
}
