package main

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/victorgama/howe/internal/updater"
)

var (
	updateDryRun bool
	updateTag    string
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update howe to the latest release",
	Long: `Checks GitHub for a newer release, downloads the appropriate binary
for the current platform, verifies its checksum, and replaces the running binary.`,
	RunE: runUpdate,
}

func init() {
	updateCmd.Flags().BoolVar(&updateDryRun, "dry-run", false, "check for updates but do not install")
	updateCmd.Flags().StringVar(&updateTag, "tag", "", "install a specific release tag instead of latest")
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	u := &updater.Updater{
		Owner:          "gophertribe",
		Repo:           "howe",
		CurrentVersion: version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := u.Update(ctx, updateTag, updateDryRun); err != nil {
		return err
	}
	return nil
}
