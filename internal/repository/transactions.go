package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morikeli/helapay-api/internal/db/store"
	"github.com/morikeli/helapay-api/internal/models"
	"github.com/morikeli/helapay-api/internal/pkg/hashutil"
)

type TransactionRepository interface {
	MakeTransaction(context.Context, models.TransactionParams) (*models.TransactionResults, bool, error)
	GetWalletBalance(context.Context, string) (*models.WalletBalance, error)
	ListWalletTransactions(context.Context, string, int, string) (*models.TransactionHistory, error)
}

type PostgresTransactionRepository struct{ pool *pgxpool.Pool }

func NewTransactionRepository(pool *pgxpool.Pool) *PostgresTransactionRepository {
	return &PostgresTransactionRepository{pool: pool}
}

// Helper to parse string UUID into pgtype.UUID using pgtype's scanner
func parseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	err := u.Scan(s)
	return u, err
}

func (r *PostgresTransactionRepository) MakeTransaction(ctx context.Context, txnParams models.TransactionParams) (*models.TransactionResults, bool, error) {
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
	if _, err := tx.Exec(ctx, `SET LOCAL lock_timeout = '1500ms'`); err != nil {
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

	// handle previously submitted payment requests
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
		var cachedResponse models.TransactionResults
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

	// Deduct creditor wallet
	debitRows, err := qtx.DebitWallet(ctx, store.DebitWalletParams{
		Balance: txnParams.Amount,
		ID:      creditorUUID,
	})
	if err != nil {
		return nil, false, fmt.Errorf("debit source wallet: %w", err)
	}
	// On a successful debit, PostgreSQL successfully updates the row and returns 1
	// If the wallet has insufficient funds, the condition fails and returns 0
	if debitRows != 1 {
		return nil, false, ErrInsufficientFunds
	}

	// create a double-entry ledger transaction
	err = qtx.CreateTransferPostings(ctx, store.CreateTransferPostingsParams{
		TransactionID: transactionID,
		WalletID:      creditorUUID,
		Amount:        txnParams.Amount,
		WalletID_2:    debtorUUID,
	})
	if err != nil {
		return nil, false, fmt.Errorf("create transfer postings: %w", err)
	}

	// credit debtor's (destination) wallet
	creditRows, err := qtx.CreditWallet(ctx, store.CreditWalletParams{
		Balance: txnParams.Amount,
		ID:      debtorUUID,
	})
	if err != nil {
		return nil, false, fmt.Errorf("credit destination wallet: %w", err)
	}
	if creditRows != 1 {
		return nil, false, errors.New("failed to credit destination wallet")
	}

	// validate transaction uuid is valid
	if !transactionID.Valid {
		return nil, false, errors.New("invalid transaction ID returned by database")
	}

	// Convert PostgreSQL UUID to string.
	txIDStr := transactionID.String()

	// Prepare domain transaction result.
	result := &models.TransactionResults{
		Status:        "COMPLETED",
		TransactionID: txIDStr,
		ReferenceID:   referenceID,
		Amount:        txnParams.Amount,
		Currency:      txnParams.Currency,
	}

	return result, false, nil

}

func (r *PostgresTransactionRepository) GetWalletBalance(ctx context.Context, walletID string) (*models.WalletBalance, error) {
	wUUID, err := parseUUID(walletID)
	if err != nil {
		return nil, ErrWalletNotFound
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	qtx := store.New(tx)

	walletRow, err := qtx.GetWalletByID(ctx, wUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWalletNotFound
	}
	if err != nil {
		return nil, err
	}

	idStr := walletRow.ID.String()

	result := &models.WalletBalance{
		WalletID: idStr,
		Balance:  walletRow.Balance,
		Currency: walletRow.Currency,
	}

	history, err := listWalletTransactions(ctx, qtx, wUUID, 10, "")
	if err != nil {
		return nil, err
	}
	result.RecentTransactions = history.Items

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

// fetch paginated results for a specific wallet
func (r *PostgresTransactionRepository) ListWalletTransactions(ctx context.Context, walletID string, limit int, cursor string) (*models.TransactionHistory, error) {
	wUUID, err := parseUUID(walletID)
	if err != nil {
		return nil, ErrWalletNotFound
	}

	qtx := store.New(r.pool)
	_, err = qtx.GetWalletByID(ctx, wUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWalletNotFound
	}
	if err != nil {
		return nil, err
	}

	return listWalletTransactions(ctx, qtx, wUUID, limit, cursor)
}

func listWalletTransactions(ctx context.Context, q *store.Queries, walletID pgtype.UUID, limit int, cursor string) (*models.TransactionHistory, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}

	postings, err := q.ListPostingsByWalletID(ctx, store.ListPostingsByWalletIDParams{
		WalletID: walletID,
		Limit:    int32(limit),
	})
	if err != nil {
		return nil, err
	}

	result := &models.TransactionHistory{
		Items: make([]models.PostingItem, 0, len(postings)),
	}

	for _, item := range postings {
		var pID, tID string
		_ = item.ID.Scan(&pID)
		_ = item.TransactionID.Scan(&tID)

		result.Items = append(result.Items, models.PostingItem{
			PostingID:     pID,
			TransactionID: tID,
			Amount:        item.Amount,
			Direction:     item.Direction,
			CreatedAt:     item.CreatedAt.Time.UTC().Format(time.RFC3339Nano),
		})
	}

	return result, nil
}
