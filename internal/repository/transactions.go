package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransactionRepository interface{}

type PostgresTransactionRepository struct{ pool *pgxpool.Pool }

func NewPostgresTransactionRepository(pool *pgxpool.Pool) *PostgresTransactionRepository {
	return &PostgresTransactionRepository{pool: pool}
}
