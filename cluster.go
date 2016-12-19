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
	"io/ioutil"

	"github.com/spf13/cobra"

	fg "github.com/pulcy/j2/flags"
)

var (
	clusterCmd = &cobra.Command{
		Use:   "cluster",
		Short: "Cluster commands.",
		Run:   func(cmd *cobra.Command, args []string) { cmd.Usage() },
	}
	clusterConfigCmd = &cobra.Command{
		Use:   "configure",
		Short: "Prepare a cluster for use by J2 jobs",
		Run:   configureClusterRun,
	}
	clusterFlags struct {
		fg.Flags

		clusterID       string
		vaultAddress    string
		vaultCACertPath string
	}
)

func init() {
	initDeploymentFlags(clusterConfigCmd.Flags(), &clusterFlags.Flags)

	fs := clusterConfigCmd.Flags()
	fs.StringVar(&clusterFlags.clusterID, "cluster-id", "", "ID of the cluster")
	fs.StringVar(&clusterFlags.vaultAddress, "vault-address", "", "URL of Vault server")
	fs.StringVar(&clusterFlags.vaultCACertPath, "vault-cacert-path", "", "Path of file containing Vault CA certificate")

	cmdMain.AddCommand(clusterCmd)
	clusterCmd.AddCommand(clusterConfigCmd)
}

func configureClusterRun(cmd *cobra.Command, args []string) {
	if clusterFlags.clusterID == "" {
		Exitf("--cluster-id missing")
	}
	if clusterFlags.vaultAddress == "" {
		Exitf("--vault-address missing")
	}
	if clusterFlags.vaultCACertPath == "" {
		Exitf("--vault-cacert-path missing")
	}

	deploymentDefaults(cmd.Flags(), &clusterFlags.Flags, args)
	runValidators(&clusterFlags.Flags)

	cluster, err := loadCluster(&clusterFlags.Flags)
	if err != nil {
		Exitf("Cannot load cluster: %v\n", err)
	}
	orchestrator, err := getOrchestrator(cluster)
	if err != nil {
		Exitf("Cannot initialize orchestrator: %v\n", err)
	}
	job, err := loadJob(&clusterFlags.Flags, *cluster, orchestrator)
	if err != nil {
		Exitf("Cannot load job: %v\n", err)
	}

	scheduler, err := orchestrator.Scheduler(*job, *cluster)
	if err != nil {
		Exitf("Cannot load scheduler: %v\n", err)
	}

	if err := scheduler.ConfigureCluster(&clusterConfig{}); err != nil {
		Exitf("Failed to configure cliuster: %#v\n", err)
	}
}

type clusterConfig struct{}

func (c *clusterConfig) ClusterID() string {
	return clusterFlags.clusterID
}

func (c *clusterConfig) VaultAddress() string {
	return clusterFlags.vaultAddress
}

func (c *clusterConfig) VaultCACert() string {
	raw, err := ioutil.ReadFile(clusterFlags.vaultCACertPath)
	if err != nil {
		Exitf("Failed to read %s: %#v\n", clusterFlags.vaultCACertPath, err)
	}
	return string(raw)
}

func (c *clusterConfig) VaultCAPath() string {
	return ""
}
