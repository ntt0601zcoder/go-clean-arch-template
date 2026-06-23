// Command account-service is the single binary for the template. Subcommands
// (server, worker, migrate) — defined in the cmd package — select which process
// to run, e.g. `go run main.go server`.
package main

import (
	"os"

	"github.com/ntt0601zcoder/go-clean-arch-template/cmd"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/logger"
)

func main() {
	if err := cmd.Execute(); err != nil {
		logger.NewSlogger("main", "info").Error("application exited with error", "err", err)
		os.Exit(1)
	}
}
