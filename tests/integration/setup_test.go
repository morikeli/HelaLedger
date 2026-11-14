package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morikeli/helapay-api/internal/models"
	"github.com/morikeli/helapay-api/internal/repository"
)

const (
	testSourceWallet      = "11111111-1111-4111-8111-111111111111"
	testDestinationWallet = "22222222-2222-4222-8222-222222222222"
)

// newIntegrationRepository creates a dedicated PostgreSQL schema for each test.
// HELAPAY_TEST_DATABASE_URL must point to a disposable test database.
func newIntegrationRepository(t *testing.T) (*repository.PostgresTransactionRepository, *pgxpool.Pool) {
	t.Helper()

	databaseURL := os.Getenv("HELAPAY_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set HELAPAY_TEST_DATABASE_URL to run PostgreSQL integration tests")
	}

	setupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	admin, err := pgx.Connect(setupCtx, databaseURL)
	if err != nil {
		t.Fatalf("connect to integration database: %v", err)
	}
	t.Cleanup(func() {
		_ = admin.Close(context.Background())
	})

	// The schema name is generated locally and contains only safe characters.
	schemaName := fmt.Sprintf("helapay_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(setupCtx, "CREATE SCHEMA "+schemaName); err != nil {
		t.Fatalf("create isolated schema: %v", err)
	}
	t.Cleanup(func() {
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer dropCancel()
		_, _ = admin.Exec(dropCtx, "DROP SCHEMA "+schemaName+" CASCADE")
	})

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse integration database URL: %v", err)
	}
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schemaName

	pool, err := pgxpool.NewWithConfig(setupCtx, poolConfig)
	if err != nil {
		t.Fatalf("create integration connection pool: %v", err)
	}
	t.Cleanup(pool.Close)

	schema, err := os.ReadFile("../../db/schema.sql")
	if err != nil {
		t.Fatalf("read database schema from ../../db/schema.sql: %v", err)
	}
	if _, err := pool.Exec(setupCtx, string(schema)); err != nil {
		t.Fatalf("apply database schema: %v", err)
	}

	_, err = pool.Exec(setupCtx, `
		INSERT INTO wallets (id, owner_id, currency, balance, status) VALUES
		($1, gen_random_uuid(), 'KES', 10000, 'ACTIVE'),
		($2, gen_random_uuid(), 'KES', 1000, 'ACTIVE')`,
		testSourceWallet, testDestinationWallet,
	)
	if err != nil {
		t.Fatalf("seed integration wallets: %v", err)
	}

	return repository.NewTransactionRepository(pool), pool
}

func transferParams(key string, amount int64) models.TransactionParams {
	return models.TransactionParams{
		IdempotencyKey:      key,
		SourceWalletID:      testSourceWallet,
		DestinationWalletID: testDestinationWallet,
		Amount:              amount,
		Currency:            "KES",
	}
}

func readWalletBalance(t *testing.T, ctx context.Context, pool *pgxpool.Pool, walletID string) int64 {
	t.Helper()

	var balance int64
	if err := pool.QueryRow(ctx, "SELECT balance FROM wallets WHERE id=$1", walletID).Scan(&balance); err != nil {
		t.Fatalf("read wallet balance for %s: %v", walletID, err)
	}
	return balance
}

func readRecordCounts(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (transactions, postings int) {
	t.Helper()

	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM transactions").Scan(&transactions); err != nil {
		t.Fatalf("count transactions: %v", err)
	}
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM postings").Scan(&postings); err != nil {
		t.Fatalf("count postings: %v", err)
	}
	return transactions, postings
}
