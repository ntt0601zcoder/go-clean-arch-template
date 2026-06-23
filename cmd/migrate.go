package cmd

import (
	"github.com/spf13/cobra"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/apps/migrate"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Apply database migrations and exit",
	RunE: func(_ *cobra.Command, _ []string) error {
		return migrate.Run()
	},
}
