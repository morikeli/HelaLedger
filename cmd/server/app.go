package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morikeli/helapay-api/internal/config"
	"github.com/morikeli/helapay-api/internal/db"
	"github.com/morikeli/helapay-api/internal/db/store"
)

type App struct {
	server   *http.Server
	database *pgxpool.Pool
}

func NewApp(cfg *config.Config) (*App, error) {
	initCtx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel()

	database, err := db.ConnectDB(initCtx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize db: %w \n", err)
	}

	// server address
	serverAddr := fmt.Sprintf(":%s", cfg.ServerPort)
	server := &http.Server{
		Addr: serverAddr,
		ReadTimeout:  10 * time.Second, // Timeout for reading request headers & body
		WriteTimeout: 10 * time.Second, // Timeout for writing response
		IdleTimeout:  time.Minute,      // Timeout for keep-alive connections
	}

	app := &App{
		server: server,
		database: database,
	}

	return app, nil
}