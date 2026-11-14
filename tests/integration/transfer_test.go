package integration

import (
	"context"
	"testing"

	"github.com/morikeli/helapay-api/internal/repository"
)

func TestMakeTransaction_HappyPathCreatesBalancedLedger(t *testing.T) {
	repo, pool := newIntegrationRepository(t)
	ctx := context.Background()

	result, replayed, err := repo.MakeTransaction(ctx, transferParams(
		"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", 2500,
	))
	if err != nil {
		t.Fatalf("MakeTransaction() error = %v", err)
	}
	if replayed {
		t.Fatal("first transfer unexpectedly marked as replay")
	}
	if result == nil || result.Status != "COMPLETED" || result.HTTPStatus != 201 {
		t.Fatalf("unexpected transfer result: %+v", result)
	}

	if got := readWalletBalance(t, ctx, pool, testSourceWallet); got != 7500 {
		t.Errorf("source balance = %d, want 7500", got)
	}
	if got := readWalletBalance(t, ctx, pool, testDestinationWallet); got != 3500 {
		t.Errorf("destination balance = %d, want 3500", got)
	}

	var debitCount, creditCount int
	var debitTotal, creditTotal int64
	err = pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE direction = 'DEBIT'),
			COUNT(*) FILTER (WHERE direction = 'CREDIT'),
			COALESCE(SUM(amount) FILTER (WHERE direction = 'DEBIT'), 0),
			COALESCE(SUM(amount) FILTER (WHERE direction = 'CREDIT'), 0)
		FROM postings
		WHERE transaction_id = $1`,
		result.TransactionID,
	).Scan(&debitCount, &creditCount, &debitTotal, &creditTotal)
	if err != nil {
		t.Fatalf("query ledger postings: %v", err)
	}

	if debitCount != 1 || creditCount != 1 {
		t.Errorf("posting counts: debit=%d credit=%d; want one of each", debitCount, creditCount)
	}
	if debitTotal != 2500 || creditTotal != 2500 {
		t.Errorf("posting totals: debit=%d credit=%d; want 2500 each", debitTotal, creditTotal)
	}
}

func TestMakeTransaction_InsufficientFundsDoesNotMutateWallets(t *testing.T) {
	repo, pool := newIntegrationRepository(t)
	ctx := context.Background()

	_, _, err := repo.MakeTransaction(ctx, transferParams(
		"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", 10001,
	))
	if err != repository.ErrInsufficientFunds {
		t.Fatalf("MakeTransaction() error = %v, want ErrInsufficientFunds", err)
	}

	if got := readWalletBalance(t, ctx, pool, testSourceWallet); got != 10000 {
		t.Errorf("source balance changed to %d after rejected transfer; want 10000", got)
	}
	if got := readWalletBalance(t, ctx, pool, testDestinationWallet); got != 1000 {
		t.Errorf("destination balance changed to %d after rejected transfer; want 1000", got)
	}

	transactions, postings := readRecordCounts(t, ctx, pool)
	if transactions != 0 || postings != 0 {
		t.Errorf("rejected transfer persisted transactions=%d postings=%d; want both zero", transactions, postings)
	}
}
