// Copyright (c) 2016 Pulcy.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jobs

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pulcy/j2/units"
)

const (
	secretsPath = "/tmp/secrets"
)

// createSecretsUnit creates a unit used to extract secrets from vault
func (t *Task) createSecretsUnit(ctx generatorContext) (*units.Unit, error) {
	// Create all secret extraction commands
	jobID := t.group.job.ID
	if jobID == "" {
		return nil, maskAny(fmt.Errorf("job ID missing for job %s with secrets", t.group.job.Name))
	}
	env := make(map[string]string)
	addArg := func(arg string, cmd *[]string) {
		if strings.Contains(arg, "$") {
			*cmd = append(*cmd, arg)
		} else {
			key := fmt.Sprintf("A%02d", len(env))
			env[key] = arg
			*cmd = append(*cmd, fmt.Sprintf("$%s", key))
		}
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
				"-v", "${SCROOT}",
				"-v", "${VOLCRT}",
				"-v", "${VOLCLS}",
				"-v", "${VOLMAC}",
				"--env-file", "/etc/pulcy/vault.env",
				ctx.Images.VaultMonkey,
				"extract",
				"file",
			}
			addArg("--target "+targetPath, &cmd)
			addArg("--job-id "+jobID, &cmd)
			addArg(secret.VaultPath(), &cmd)
			cmds = append(cmds, cmd)
		} else if ok, environmentKey := secret.TargetEnviroment(); ok {
			envPaths = append(envPaths, fmt.Sprintf("%s=%s", environmentKey, secret.VaultPath()))
		}
	}
	if len(envPaths) > 0 {
		targetPath := t.secretEnvironmentPath(ctx.ScalingGroup)
		cmd := []string{
			"/usr/bin/docker",
			"run",
			"--rm",
			"-v", "${SCROOT}",
			"-v", "${VOLCRT}",
			"-v", "${VOLCLS}",
			"-v", "${VOLMAC}",
			"--env-file", "/etc/pulcy/vault.env",
			ctx.Images.VaultMonkey,
			"extract",
			"env",
		}
		addArg("--target "+targetPath, &cmd)
		addArg("--job-id "+jobID, &cmd)
		for _, envPath := range envPaths {
			addArg(envPath, &cmd)
		}
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
	secretsRoot := t.secretsRootPath(ctx.ScalingGroup)
	unit.ExecOptions.Environment["SCROOT"] = fmt.Sprintf("%s:%s", secretsRoot, secretsRoot)
	unit.ExecOptions.Environment["VOLCRT"] = "/etc/pulcy/vault.crt:/etc/pulcy/vault.crt:ro"
	unit.ExecOptions.Environment["VOLCLS"] = "/etc/pulcy/cluster-id:/etc/pulcy/cluster-id:ro"
	unit.ExecOptions.Environment["VOLMAC"] = "/etc/machine-id:/etc/machine-id:ro"
	for k, v := range env {
		unit.ExecOptions.Environment[k] = v
	}

	unit.ExecOptions.IsOneshot = true
	unit.ExecOptions.ExecStartPre = append(unit.ExecOptions.ExecStartPre,
		fmt.Sprintf("/usr/bin/mkdir -p %s", secretsRoot),
		fmt.Sprintf("/usr/bin/docker pull %s", ctx.Images.VaultMonkey),
	)
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
	return filepath.Join(secretsPath, t.containerName(scalingGroup))
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
