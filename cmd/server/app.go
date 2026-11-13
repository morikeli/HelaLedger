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

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morikeli/helapay-api/internal/config"
	"github.com/morikeli/helapay-api/internal/db"
	"github.com/morikeli/helapay-api/internal/handlers"
	"github.com/morikeli/helapay-api/internal/repository"
	"github.com/morikeli/helapay-api/internal/services"
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

	txnRepo := repository.NewTransactionRepository(database)
	txnService := services.NewTransactionService(txnRepo)
	txnHandler := handlers.NewTransactionHandler(txnService)

	// initiliaze Chi
	r := chi.NewRouter()

	// register middlewares using Chi
	r.Use(middleware.RequestID)
	r.Use(middleware.ClientIPFromRemoteAddr)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(60 * time.Second))

	// handlers
	r.Get("/health/live", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("We are live!\n"))
	})
	r.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := database.Ping(r.Context()); err != nil {
			http.Error(w, "Oops! Service unavailable.", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Ready for take off!\n"))
	})
	
	r.Route("/v1", func(r chi.Router) {
		r.Post("/transfers", txnHandler.CreateTransactionHandler)
		r.Get("/ledger/check-balance", txnHandler.CheckLedgerBalanceHandler)
		r.Get("/wallets/{id}/balance", txnHandler.GetWalletBalanceHandler)
		r.Get("/wallets/{id}/transactions", txnHandler.GetTransactionHistoryHandler)
	})

	// server address
	serverAddr := fmt.Sprintf(":%s", cfg.ServerPort)
	server := &http.Server{
		Addr:         serverAddr,
		Handler: r,
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
		slog.Info("Server starting on", "port", port)
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
