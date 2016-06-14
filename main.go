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

	"github.com/op/go-logging"
	"github.com/spf13/cobra"

	"github.com/pulcy/j2/jobs/render"
)

const (
	projectName = "j2"
)

var (
	projectVersion = "dev"
	projectBuild   = "dev"
	images         = render.Images{
		VaultMonkey: "pulcy/vault-monkey:latest",
		Wormhole:    "pulcy/wormhole:latest",
		Alpine:      "alpine:3.3",
		CephVolume:  "pulcy/ceph-volume:latest",
	}
)

var (
	cmdMain = &cobra.Command{
		Use: projectName,
		Run: showUsage,
		PersistentPreRun: func(*cobra.Command, []string) {
			setLogLevel(globalFlags.logLevel, defaultLogLevel, projectName)
			setLogLevel(globalFlags.fleetLogLevel, globalFlags.logLevel, "fleet")
		},
	}
	globalFlags struct {
		debug         bool
		verbose       bool
		logLevel      string
		fleetLogLevel string
	}
	log *logging.Logger
)

func init() {
	log = logging.MustGetLogger(projectName)
	cmdMain.PersistentFlags().BoolVarP(&globalFlags.debug, "debug", "D", false, "Print debug output")
	cmdMain.PersistentFlags().BoolVarP(&globalFlags.verbose, "verbose", "v", false, "Print verbose output")
	cmdMain.PersistentFlags().StringVar(&globalFlags.logLevel, "log-level", defaultLogLevel, "Log level (debug|info|warning|error)")
	cmdMain.PersistentFlags().StringVar(&globalFlags.fleetLogLevel, "fleet-log-level", "", "Log level of the fleet tunnel (debug|info|warning|error)")
}

func main() {
	cmdMain.AddCommand(runCmd)
	cmdMain.AddCommand(destroyCmd)

	cmdMain.Execute()
}

func showUsage(cmd *cobra.Command, args []string) {
	cmd.Usage()
}

func Exitf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	fmt.Println()
	os.Exit(1)
}

func assert(err error) {
	if err != nil {
		Exitf("Assertion failed: %#v", err)
	}
}

func setLogLevel(logLevel, defaultLogLevel, logName string) {
	if logLevel == "" {
		logLevel = defaultLogLevel
	}
	level, err := logging.LogLevel(logLevel)
	if err != nil {
		Exitf("Invalid log-level '%s': %#v", logLevel, err)
	}
	logging.SetLevel(level, logName)
}
