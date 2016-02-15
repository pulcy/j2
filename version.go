package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	hdr = `   ___       __              _____
  / _ \__ __/ /_____ __  __ / /_  |
 / ___/ // / / __/ // / / // / __/
/_/   \_,_/_/\__/\_, /  \___/____/
                /___/
`
)

var (
	cmdVersion = &cobra.Command{
		Use: "version",
		Run: showVersion,
	}
)

func init() {
	cmdMain.AddCommand(cmdVersion)
}

func showVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("%s\n", hdr)
	fmt.Printf("%s %s, build %s\n", cmdMain.Use, projectVersion, projectBuild)
}
