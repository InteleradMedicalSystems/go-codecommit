package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version string
var GitCommit string

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version info",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Version %s, GitCommit %s\n", Version, GitCommit)
		},
	}
}
