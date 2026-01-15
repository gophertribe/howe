package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of howe",
	Long:  `Print the version number and build information of howe.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("howe version %s\n", version)
		if commit != "unknown" {
			fmt.Printf("commit: %s\n", commit)
		}
		if date != "unknown" {
			fmt.Printf("built: %s\n", date)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
