package cmd

import (
	"github.com/spf13/cobra"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/apps/worker"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Run the background worker (Kafka consumer + scheduled jobs)",
	RunE: func(_ *cobra.Command, _ []string) error {
		worker.Start()
		return nil
	},
}
