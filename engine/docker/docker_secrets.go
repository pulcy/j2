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

package docker

import (
	"fmt"
	"path/filepath"

	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/pkg/cmdline"
)

const (
	secretsPath = "/tmp/secrets"
)

// createSecretsUnit creates a unit used to extract secrets from vault
func (e *dockerEngine) createSecretsExecStartPre(t *jobs.Task, containerImage string, env map[string]string, scalingGroup uint) ([]cmdline.Cmdline, error) {
	if len(t.Secrets) == 0 {
		// No secrets to extract
		return nil, nil
	}
	// Create all secret extraction commands
	jobID := t.JobID()
	if jobID == "" {
		return nil, maskAny(fmt.Errorf("job ID missing for job %s with secrets", t.JobName()))
	}

	// Prepare volume paths
	secretsRoot := secretsRootPath(t, scalingGroup)
	secretsRootVol := fmt.Sprintf("%s:%s", secretsRoot, secretsRoot)
	vaultCrtVol := "/etc/pulcy/vault.crt:/etc/pulcy/vault.crt:ro"
	clusterIdVol := "/etc/pulcy/cluster-id:/etc/pulcy/cluster-id:ro"
	machineIdVol := "/etc/machine-id:/etc/machine-id:ro"

	var cmds []cmdline.Cmdline
	cmds = append(cmds,
		*cmdline.New(nil, "/usr/bin/mkdir", "-p", secretsRoot),
		e.pullCmd(containerImage),
	)
	envPaths := []string{}
	for _, secret := range t.Secrets {
		if ok, _ := secret.TargetFile(); ok {
			targetPath, err := secretFilePath(t, scalingGroup, secret)
			if err != nil {
				return nil, maskAny(err)
			}
			var cmd cmdline.Cmdline
			cmd.Add(nil, e.dockerPath, "run", "--rm")
			//cmd.Add(env, fmt.Sprintf("--name %s-sc", t.containerName(ctx.ScalingGroup)))
			cmd.Add(env, "--net=host")
			cmd.Add(env, "-v "+secretsRootVol)
			cmd.Add(env, "-v "+vaultCrtVol)
			cmd.Add(env, "-v "+clusterIdVol)
			cmd.Add(env, "-v "+machineIdVol)
			cmd.Add(env, "--env-file /etc/pulcy/vault.env")
			/*if ctx.DockerOptions.EnvFile != "" {
				cmd.Add(env,fmt.Sprintf("--env-file=%s", ctx.DockerOptions.EnvFile))
			}*/
			for _, arg := range t.LogDriver.CreateDockerLogArgs(e.options) {
				cmd.Add(env, arg)
			}
			cmd.Add(env, containerImage)
			cmd.Add(nil, "extract", "file")
			cmd.Add(env, "--target "+targetPath)
			cmd.Add(env, "--job-id "+jobID)
			cmd.Add(env, secret.VaultPath())
			cmds = append(cmds, cmd)
		} else if ok, environmentKey := secret.TargetEnviroment(); ok {
			envPaths = append(envPaths, fmt.Sprintf("%s=%s", environmentKey, secret.VaultPath()))
		}
	}
	if len(envPaths) > 0 {
		targetPath := secretEnvironmentPath(t, scalingGroup)
		var cmd cmdline.Cmdline
		cmd.Add(nil, e.dockerPath, "run", "--rm")
		//cmd.Add(env, fmt.Sprintf("--name %s-sc", t.containerName(ctx.ScalingGroup)))
		cmd.Add(env, "--net=host")
		cmd.Add(env, "-v "+secretsRootVol)
		cmd.Add(env, "-v "+vaultCrtVol)
		cmd.Add(env, "-v "+clusterIdVol)
		cmd.Add(env, "-v "+machineIdVol)
		cmd.Add(env, "--env-file /etc/pulcy/vault.env")
		/*if ctx.DockerOptions.EnvFile != "" {
			cmd.Add(env, fmt.Sprintf("--env-file=%s", ctx.DockerOptions.EnvFile))
		}*/
		for _, arg := range t.LogDriver.CreateDockerLogArgs(e.options) {
			cmd.Add(env, arg)
		}
		cmd.Add(env, containerImage)
		cmd.Add(nil, "extract", "env")
		cmd.Add(env, "--target "+targetPath)
		cmd.Add(env, "--job-id "+jobID)
		for _, envPath := range envPaths {
			cmd.Add(env, envPath)
		}
		cmds = append(cmds, cmd)
	}

	return cmds, nil
}

// secretsRootPath returns the path of the root directory that will contain secret files for the given task.
func secretsRootPath(t *jobs.Task, scalingGroup uint) string {
	return filepath.Join(secretsPath, t.ContainerName(scalingGroup))
}

// secretEnvironmentPath returns the path of the file containing all secret environment variables
// for the given container.
func secretEnvironmentPath(t *jobs.Task, scalingGroup uint) string {
	return filepath.Join(secretsRootPath(t, scalingGroup), "environment")
}

// secretFilePath returns the path of the file containing the given secret (file type)
func secretFilePath(t *jobs.Task, scalingGroup uint, secret jobs.Secret) (string, error) {
	if secret.File == "" {
		return "", maskAny(fmt.Errorf("Wrong secret, file must be non-empty"))
	}
	hash, err := secret.Hash()
	if err != nil {
		return "", maskAny(err)
	}
	return filepath.Join(secretsRootPath(t, scalingGroup), hash), nil
}
