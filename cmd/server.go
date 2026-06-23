package cmd

import (
	"github.com/spf13/cobra"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/apps/server"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the HTTP API + gRPC server",
	RunE: func(_ *cobra.Command, _ []string) error {
		server.Start()
		return nil
	},
}
