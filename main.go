package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"arvika.pulcy.com/pulcy/deployit/jobs"
)

var (
	projectVersion = "dev"
	projectBuild   = "dev"
	images         = jobs.Images{
		VaultMonkey: "pulcy/vault-monkey:latest",
	}
)

var (
	cmdMain = &cobra.Command{
		Use:              "deployit",
		Run:              showUsage,
		PersistentPreRun: loadDefaults,
	}
	globalFlags struct {
		debug   bool
		verbose bool
	}
)

func init() {
	cmdMain.PersistentFlags().BoolVarP(&globalFlags.debug, "debug", "D", false, "Print debug output")
	cmdMain.PersistentFlags().BoolVarP(&globalFlags.verbose, "verbose", "v", false, "Print verbose output")
}

func main() {
	cmdMain.AddCommand(runCmd)
	cmdMain.AddCommand(destroyCmd)

	cmdMain.Execute()
}

func showUsage(cmd *cobra.Command, args []string) {
	cmd.Usage()
}

func loadDefaults(cmd *cobra.Command, args []string) {
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
