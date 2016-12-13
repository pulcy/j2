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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/juju/errgo"
	"github.com/kardianos/osext"
	"github.com/spf13/pflag"

	"github.com/pulcy/j2/cluster"
	fg "github.com/pulcy/j2/flags"
	"github.com/pulcy/j2/jobs"
)

func initDeploymentFlags(fs *pflag.FlagSet, f *fg.Flags) {
	fs.StringVarP(&f.JobPath, "job", "j", defaultJobPath, "filename of the job description")
	fs.StringVarP(&f.ClusterPath, "cluster", "c", defaultClusterPath, "cluster description name or filename")
	fs.StringVar(&f.OrchestratorOverride, "orchestrator", defaultOrchestratorOverride, "Type of orchestrator to use. This overrides cluster or default settings")
	fs.StringVarP(&f.TunnelOverride, "tunnel", "t", defaultTunnelOverride, "SSH endpoint to tunnel through with fleet (cluster override)")
	fs.StringSliceVarP(&f.Groups, "groups", "g", defaultGroups, "target task groups to deploy")
	fs.BoolVarP(&f.Force, "force", "f", defaultForce, "wheather to confirm destroy or not")
	fs.BoolVarP(&f.AutoContinue, "yes", "y", false, "auto continue on confirmation")
	fs.BoolVarP(&f.DryRun, "dry-run", "d", defaultDryRun, "wheather to schedule units or not")
	fs.UintVar(&f.ScalingGroup, "scaling-group", defaultScalingGroup, "scaling group to deploy")
	fs.BoolVarP(&f.Local, "local", "l", defaultLocal, "User local vagrant test cluster")
	fs.DurationVar(&f.StopDelay, "stop-delay", defaultStopDelay, "Time between stop and destroy")
	fs.DurationVar(&f.DestroyDelay, "destroy-delay", defaultDestroyDelay, "Time between destroy and re-create")
	fs.DurationVar(&f.SliceDelay, "slice-delay", defaultSliceDelay, "Time between update of scaling slices")
	fs.VarP(&f.Options, "option", "o", "Set an option (key=value)")

	f.VaultCACert = os.Getenv("VAULT_CACERT")
	f.VaultCAPath = os.Getenv("VAULT_CAPATH")
	fs.StringVar(&f.VaultAddr, "vault-addr", "", "URL of the vault (defaults to VAULT_ADDR environment variable)")
	fs.StringVar(&f.VaultCACert, "vault-cacert", f.VaultCACert, "Path to a PEM-encoded CA cert file to use to verify the Vault server SSL certificate")
	fs.StringVar(&f.VaultCAPath, "vault-capath", f.VaultCAPath, "Path to a directory of PEM-encoded CA cert files to verify the Vault server SSL certificate")
	fs.StringVarP(&f.GithubToken, "github-token", "G", "", "Personal github token for secret logins")
	fs.StringVar(&f.GithubTokenPath, "github-token-path", defaultGithubTokenPath, "Path of a file containing your github token")
}

func deploymentDefaults(fs *pflag.FlagSet, f *fg.Flags, args []string) {
	// Merge Options
	fs.VisitAll(func(flag *pflag.Flag) {
		if !flag.Changed {
			value, ok := f.Options.Get(flag.Name)
			if ok {
				svalue := fmt.Sprintf("%v", value)
				err := fs.Set(flag.Name, svalue)
				if err != nil {
					Exitf("Error in option '%s': %#v\n", flag.Name, err)
				}
			}
		}
	})

	if f.Local {
		f.StopDelay = 5 * time.Second
		f.DestroyDelay = 3 * time.Second
		f.SliceDelay = 5 * time.Second
	}

	if f.JobPath == "" && len(args) >= 1 {
		f.JobPath = args[0]
	}
	if f.ClusterPath == "" && len(args) >= 2 {
		f.ClusterPath = args[1]
	}
}

// Gets the list of group names to operate on based on the deployment flags.
func groups(f *fg.Flags) []jobs.TaskGroupName {
	names := []jobs.TaskGroupName{}
	for _, n := range f.Groups {
		names = append(names, jobs.TaskGroupName(n))
	}
	return names
}

// loadJob loads the a job from the given flags.
func loadJob(f *fg.Flags, cluster cluster.Cluster) (*jobs.Job, error) {
	if f.JobPath == "" {
		return nil, maskAny(errgo.New("--job missing"))
	}
	path, err := resolvePath(f.JobPath, "config", ".hcl")
	if err != nil {
		return nil, maskAny(err)
	}
	job, err := jobs.ParseJobFromFile(path, cluster, f.Options, log, f.VaultConfig, f.GithubLoginData)
	if err != nil {
		return nil, maskAny(err)
	}
	return job, nil
}

// loadCluster loads a cluster description from the given flags.
func loadCluster(f *fg.Flags) (*cluster.Cluster, error) {
	if f.ClusterPath == "" {
		return nil, maskAny(errgo.New("--cluster missing"))
	}
	clustersPath := os.Getenv("PULCY_CLUSTERS")
	if clustersPath == "" {
		clustersPath = "config/clusters"
	}
	path, err := resolvePath(f.ClusterPath, clustersPath, ".hcl")
	if err != nil {
		return nil, maskAny(err)
	}
	cluster, err := cluster.ParseClusterFromFile(path)
	if err != nil {
		return nil, maskAny(err)
	}
	if f.OrchestratorOverride != "" {
		cluster.Orchestrator = f.OrchestratorOverride
	}
	if f.TunnelOverride != "" {
		cluster.Tunnel = f.TunnelOverride
	}
	if f.Local {
		cluster.Tunnel = "core-01"
		cluster.Stack = "core-01"
	}
	return cluster, nil
}

// resolvePath tries to resolve a given path.
// 1) Try as real path
// 2) Try as filename relative to my process with given relative folder & extension
func resolvePath(path, altFolder, extension string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// path not found, try locating it by name in a different folder
		var folder string
		if filepath.IsAbs(altFolder) {
			folder = altFolder
		} else {
			// altFolder is relative, assume it is relative to our executable
			exeFolder, err := osext.ExecutableFolder()
			if err != nil {
				return "", maskAny(err)
			}
			folder = filepath.Join(exeFolder, altFolder)
		}
		path = filepath.Join(folder, path) + extension
		if _, err := os.Stat(path); os.IsNotExist(err) {
			// Try without extensions
			path = filepath.Join(folder, path)
		}
	}
	return path, nil
}
