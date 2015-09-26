package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	projectVersion = "dev"
	projectBuild   = "dev"
	domain         = "pulcy.com"
)

var (
	cmdMain = &cobra.Command{
		Use:              "deployit",
		Run:              showUsage,
		PersistentPreRun: loadDefaults,
	}
)

func init() {
	//cmdMain.PersistentFlags().StringVarP(&digitalOceanToken, "digitalocean-token", "t", "", "Digital Ocean token")
}

func main() {
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
	os.Exit(1)
}
