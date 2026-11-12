package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransactionRepository interface{}

type PostgresTransactionRepository struct{ pool *pgxpool.Pool }

func NewPostgresTransactionRepository(pool *pgxpool.Pool) *PostgresTransactionRepository {
	return &PostgresTransactionRepository{pool: pool}
}

// Helper to parse string UUID into pgtype.UUID using pgtype's scanner
func parseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	err := u.Scan(s)
	return u, err
}
