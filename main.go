package main

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	projectVersion = "dev"
	projectBuild   = "dev"
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
		sleep   time.Duration
	}
)

func init() {
	cmdMain.PersistentFlags().BoolVarP(&globalFlags.debug, "debug", "d", false, "Print debug output")
	cmdMain.PersistentFlags().BoolVarP(&globalFlags.verbose, "verbose", "v", false, "Print verbose output")
	cmdMain.PersistentFlags().DurationVar(&globalFlags.sleep, "sleep", 60*time.Second, "time to sleep between updates")
}

func main() {
	cmdMain.AddCommand(createCmd)
	cmdMain.AddCommand(destroyCmd)
	cmdMain.AddCommand(updateCmd)

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

		if string(line) == "yes" {
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

func assert(err error) {
	if err != nil {
		panic(err.Error())
	}
}
