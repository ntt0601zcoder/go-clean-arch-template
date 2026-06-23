// Package cmd defines the Cobra command tree. main.go calls Execute(); each
// subcommand boots one FX-wired process (see internal/apps). The root command
// only groups the subcommands and has no behaviour of its own.
package cmd

import (
	"time"

	"github.com/spf13/cobra"
)

func init() {
	// Run every process in UTC for consistent timestamps/logs.
	time.Local = time.UTC

	rootCmd.AddCommand(serverCmd, workerCmd, migrateCmd)
}

var rootCmd = &cobra.Command{
	Use:           "account-service",
	Short:         "Sample clean-architecture Go service (account domain)",
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
