package main

import (
	"log/slog"
	"os"

	"github.com/morikeli/helapay-api/cmd/server"
	"github.com/morikeli/helapay-api/internal/config"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("load configuration", "error", err)
		os.Exit(1)
	}

	application, err := server.NewApp(cfg)
	if err != nil {
		slog.Error("initialize application", "error", err)
		os.Exit(1)
	}

	if err := application.Run(cfg.ServerPort); err != nil {
		slog.Error("application stopped with error", "error", err)
		os.Exit(1)
	}
}
