package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morikeli/helapay-api/internal/db/store"
	"github.com/morikeli/helapay-api/internal/domain"
	"github.com/morikeli/helapay-api/internal/pkg/hashutil"
)

type TransactionRepository interface {
	Transact(context.Context, domain.TransactionParams) (*domain.TransactionResults, bool, error)
}

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

func (r *PostgresTransactionRepository) Transact(ctx context.Context, txnParams domain.TransactionParams) (*domain.TransactionResults, bool, error) {
	// Validate transaction
	if txnParams.SourceWalletID == txnParams.DestinationWalletID {
		return nil, false, ErrInvalidTransfer
	}

	if txnParams.Amount <= 0 {
		return nil, false, ErrInvalidTransfer
	}

	// parse idempotency string into pgtype.UUID
	idempotencyUUID, err := parseUUID(txnParams.IdempotencyKey)
	if err != nil {
		return nil, false, fmt.Errorf("invalid idempotency key format: %w", err)
	}

	hash, err := hashutil.HashPayload(txnParams)
	if err != nil {
		return nil, false, fmt.Errorf("generate payload hash: %w", err)
	}

	// begin db transaction
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, false, fmt.Errorf("begin transaction: %w", err)
	}

	// rollback if transaction hasn't been committed
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	// Bind sqlc queries to the active transaction
	qtx := store.New(tx)

	// configure db lock timeout
	if _, err := tx.Exec(ctx, `SET LOCAL timeout = '1500ms'`); err != nil {
		return nil, false, fmt.Errorf("set lock timeout: %w", err)
	}

	// atomically claim idempotency key
	rowsAffected, err := qtx.ClaimIdempotencyKey(ctx, store.ClaimIdempotencyKeyParams{IdempotencyKey: idempotencyUUID, PayloadHash: hash})
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "55P03" {
			return nil, false, ErrRequestInProgress
		}

		return nil, false, fmt.Errorf("claim idempotency key: %w", err)
	}

	if rowsAffected == 0 {
		existing, err := qtx.GetIdempotencyKey(ctx, idempotencyUUID)
		if err != nil {
			return nil, false, fmt.Errorf("retrieve idempotency record: %w", err)
		}

		if existing.PayloadHash != hash {
			return nil, false, ErrIdempotencyConflict
		}

		if existing.Status != "COMPLETED" || !existing.ResponseStatus.Valid || len(existing.ResponseBody) == 0 {
			return nil, false, ErrRequestInProgress
		}

		// handle idempotent retry - network timeout or connection drop
		var cachedResponse domain.TransactionResults
		if err := json.Unmarshal(existing.ResponseBody, &cachedResponse); err != nil {
			return nil, false, fmt.Errorf("decode cached response: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, false, fmt.Errorf("commit replay transaction: %w", err)
		}

		return &cachedResponse, true, nil
	}

	// Lock both wallets in a consistent deterministic order to avoid deadlocks
	walletIDs := []string{txnParams.SourceWalletID, txnParams.DestinationWalletID}
	sort.Strings(walletIDs)

	srcWallet, err := parseUUID(walletIDs[0])
	if err != nil {
		return nil, false, fmt.Errorf("parse source wallet UUID: %w", err)
	}

	destWallet, err := parseUUID(walletIDs[1])
	if err != nil {
		return nil, false, fmt.Errorf("parse destination wallet UUID: %w", err)
	}

	lockedWallets, err := qtx.LockTransferWallets(ctx, store.LockTransferWalletsParams{Column1: srcWallet, Column2: destWallet})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "55P03" {
			// "55P03" corresponds to the PostgreSQL "lock_not_available" error.
			// This happens when our SET LOCAL lock_timeout expires because
			// another concurrent request is currently holding a lock on the row.
			return nil, false, ErrRequestInProgress
		}
		return nil, false, fmt.Errorf("lock transfer wallets: %w", err)
	}

	walletsMap := make(map[string]store.LockTransferWalletsRow, len(lockedWallets))
	for _, w := range lockedWallets {
		walletsMap[w.ID] = w
	}

	creditor, creditorExists := walletsMap[txnParams.SourceWalletID]
	debtor, debtorExists := walletsMap[txnParams.DestinationWalletID]

	if !creditorExists || !debtorExists {
		return nil, false, ErrWalletNotFound
	}

	// validate wallet status
	if creditor.Status != "ACTIVE" || debtor.Status != "ACTIVE" {
		return nil, false, ErrWalletInactive
	}

	// validate currency
	if creditor.Currency != txnParams.Currency || debtor.Currency != txnParams.Currency {
		return nil, false, ErrCurrencyMismatch
	}

	// validate available balance
	if creditor.Balance < txnParams.Amount {
		return nil, false, ErrInsufficientFunds
	}

	// prevent integer overflow when crediting debtor (reciever)
	//
	// If a destination wallet's balance is already very large, adding the transfer amount could exceed
	// this maximum limit. In computer systems, exceeding the integer limit causes an integer overflow,
	// which wraps the number around into a negative value, corrupting financial records.
	if debtor.Balance > math.MaxInt64-txnParams.Amount {
		return nil, false, ErrBalanceOverflow
	}

	// create transaction record
	referenceID := "REF-#" + uuid.New().String()
	transactionID, err := qtx.CreateTransferTransaction(ctx, store.CreateTransferTransactionParams{
		ReferenceID: referenceID,
		Amount:      txnParams.Amount,
		Currency:    txnParams.Currency,
	})
	if err != nil {
		return nil, false, fmt.Errorf("create transfer transaction: %w", err)
	}

	creditorUUID, err := parseUUID(txnParams.SourceWalletID)
	if err != nil {
		return nil, false, fmt.Errorf("parse creditor UUID: %w", err)
	}
	debtorUUID, err := parseUUID(txnParams.DestinationWalletID)
	if err != nil {
		return nil, false, fmt.Errorf("parse debtor UUID: %w", err)
	}


}
