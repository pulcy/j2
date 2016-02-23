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
	"strings"
)

const (
	secretsPath = "/tmp/secrets"
)

// createSecretsUnit creates a unit used to extract secrets from vault
func (t *Task) createSecretsExecStartPre(env map[string]string, ctx generatorContext) ([]string, error) {
	if len(t.Secrets) == 0 {
		// No secrets to extract
		return nil, nil
	}
	// Create all secret extraction commands
	jobID := t.group.job.ID
	if jobID == "" {
		return nil, maskAny(fmt.Errorf("job ID missing for job %s with secrets", t.group.job.Name))
	}

	// Prepare volume paths
	secretsRoot := t.secretsRootPath(ctx.ScalingGroup)
	secretsRootVol := fmt.Sprintf("%s:%s", secretsRoot, secretsRoot)
	vaultCrtVol := "/etc/pulcy/vault.crt:/etc/pulcy/vault.crt:ro"
	clusterIdVol := "/etc/pulcy/cluster-id:/etc/pulcy/cluster-id:ro"
	machineIdVol := "/etc/machine-id:/etc/machine-id:ro"

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
			}
			addArg(fmt.Sprintf("--name %s-sc", t.containerName(ctx.ScalingGroup)), &cmd, env)
			addArg("-v "+secretsRootVol, &cmd, env)
			addArg("-v "+vaultCrtVol, &cmd, env)
			addArg("-v "+clusterIdVol, &cmd, env)
			addArg("-v "+machineIdVol, &cmd, env)
			addArg("--env-file /etc/pulcy/vault.env", &cmd, env)
			for _, arg := range t.LogDriver.CreateDockerLogArgs(ctx.DockerOptions) {
				addArg(arg, &cmd, env)
			}
			addArg(ctx.Images.VaultMonkey, &cmd, env)
			cmd = append(cmd,
				"extract",
				"file",
			)
			addArg("--target "+targetPath, &cmd, env)
			addArg("--job-id "+jobID, &cmd, env)
			addArg(secret.VaultPath(), &cmd, env)
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
		}
		addArg(fmt.Sprintf("--name %s-sc", t.containerName(ctx.ScalingGroup)), &cmd, env)
		addArg("-v "+secretsRootVol, &cmd, env)
		addArg("-v "+vaultCrtVol, &cmd, env)
		addArg("-v "+clusterIdVol, &cmd, env)
		addArg("-v "+machineIdVol, &cmd, env)
		addArg("--env-file /etc/pulcy/vault.env", &cmd, env)
		for _, arg := range t.LogDriver.CreateDockerLogArgs(ctx.DockerOptions) {
			addArg(arg, &cmd, env)
		}
		addArg(ctx.Images.VaultMonkey, &cmd, env)
		cmd = append(cmd,
			"extract",
			"env",
		)
		addArg("--target "+targetPath, &cmd, env)
		addArg("--job-id "+jobID, &cmd, env)
		for _, envPath := range envPaths {
			addArg(envPath, &cmd, env)
		}
		cmds = append(cmds, cmd)
	}

	// Create ExecStartPre result
	result := []string{
		fmt.Sprintf("/usr/bin/mkdir -p %s", secretsRoot),
		fmt.Sprintf("/usr/bin/docker pull %s", ctx.Images.VaultMonkey),
	}
	for _, cmd := range cmds {
		result = append(result, strings.Join(cmd, " "))
	}

	return result, nil
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
