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
	"bufio"
	"fmt"
	"os"

	"github.com/op/go-logging"
	"github.com/spf13/cobra"

	"github.com/pulcy/j2/jobs"
)

const (
	projectName = "j2"
)

var (
	projectVersion = "dev"
	projectBuild   = "dev"
	images         = jobs.Images{
		VaultMonkey: "pulcy/vault-monkey:latest",
		Wormhole:    "pulcy/wormhole:latest",
		Alpine:      "alpine:3.3",
	}
)

var (
	cmdMain = &cobra.Command{
		Use:              projectName,
		Run:              showUsage,
		PersistentPreRun: func(*cobra.Command, []string) { setLogLevel(globalFlags.logLevel) },
	}
	globalFlags struct {
		debug    bool
		verbose  bool
		logLevel string
	}
	log *logging.Logger
)

func init() {
	log = logging.MustGetLogger(projectName)
	cmdMain.PersistentFlags().BoolVarP(&globalFlags.debug, "debug", "D", false, "Print debug output")
	cmdMain.PersistentFlags().BoolVarP(&globalFlags.verbose, "verbose", "v", false, "Print verbose output")
	cmdMain.PersistentFlags().StringVar(&globalFlags.logLevel, "log-level", defaultLogLevel, "Log level (debug|info|warning|error)")
}

func main() {
	cmdMain.AddCommand(runCmd)
	cmdMain.AddCommand(destroyCmd)

	cmdMain.Execute()
}

func showUsage(cmd *cobra.Command, args []string) {
	cmd.Usage()
}

func confirm(question string) error {
	for {
		fmt.Printf("%s [yes|no]", question)
		bufStdin := bufio.NewReader(os.Stdin)
		line, _, err := bufStdin.ReadLine()
		if err != nil {
			return err
		}

		if string(line) == "yes" || string(line) == "y" {
			return nil
		}
		fmt.Println("Please enter 'yes' to confirm.")
	}
}

func Exitf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	fmt.Println()
	os.Exit(1)
}

func Verbosef(format string, args ...interface{}) {
	if globalFlags.verbose {
		fmt.Printf(format, args...)
	}
}

func assert(err error) {
	if err != nil {
		Exitf("Assertion failed: %#v", err)
	}
}

func setLogLevel(logLevel string) {
	level, err := logging.LogLevel(logLevel)
	if err != nil {
		Exitf("Invalid log-level '%s': %#v", logLevel, err)
	}
	logging.SetLevel(level, projectName)
}
