// The multi-line comments below are annotations used to generate Swagger docs
//
// @title           HelaLedger API
// @version         1.0
// @description     API documentation for HelaPay backend services.
// @termsOfService  http://swagger.io/terms/

// @contact.name    API Support
// @contact.email   support@helapay.com

// @license.name    Apache 2.0
// @license.url     http://www.apache.org/licenses/LICENSE-2.0.html

// @host            localhost:8000
// @BasePath        /v1
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
