package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
	initCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	database, err := db.ConnectDB(initCtx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize db: %w \n", err)
	}

	// server address
	serverAddr := fmt.Sprintf(":%s", cfg.ServerPort)
	server := &http.Server{
		Addr:         serverAddr,
		ReadTimeout:  10 * time.Second, // Timeout for reading request headers & body
		WriteTimeout: 10 * time.Second, // Timeout for writing response
		IdleTimeout:  time.Minute,      // Timeout for keep-alive connections
	}

	app := &App{
		server:   server,
		database: database,
	}

	return app, nil
}

func (a *App) Run(port string) error {
	// Channel to signal server startup errors
	serverErrors := make(chan error, 1)

	// Start server in a non-blocking goroutine
	go func() {
		slog.Info("Server starting on port %s...\n", "port", port)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// Listen for OS signals for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Block until a signal or server startup error is received
	select {
	case err := <-serverErrors:
		a.database.Close()
		return fmt.Errorf("Server startup failed: %w", err)

	case <-ctx.Done():
		return a.shutdown()
	}
}

func (a *App) shutdown() error {
	// Create a timeout context for the shutdown process (e.g., 10 seconds)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Drain HTTP server requests first; stop accepting new connections
	if err := a.server.Shutdown(ctx); err != nil {
		_ = a.server.Close()
		a.database.Close()
		return fmt.Errorf("HTTP server shutdown forced: %v", err)
	}

	// Close database after HTTP server has drained requests
	a.database.Close()
	slog.Info("Database connection closed successfully.")
	slog.Info("Graceful shutdown complete.")

	return nil
}
