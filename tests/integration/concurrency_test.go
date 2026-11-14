package integration

import (
	"context"
	"sync"
	"testing"

	"github.com/morikeli/helapay-api/internal/repository"
)

func TestMakeTransaction_ConcurrentDebitsCannotOverdrawWallet(t *testing.T) {
	repo, pool := newIntegrationRepository(t)
	ctx := context.Background()

	// With 6000 cents available, only one 5000-cent transfer can succeed.
	if _, err := pool.Exec(ctx,
		"UPDATE wallets SET balance=6000 WHERE id=$1", testSourceWallet,
	); err != nil {
		t.Fatalf("set source balance for concurrency test: %v", err)
	}

	keys := []string{
		"cccccccc-cccc-4ccc-8ccc-cccccccccccc",
		"dddddddd-dddd-4ddd-8ddd-dddddddddddd",
	}

	start := make(chan struct{})
	type outcome struct {
		err error
	}
	outcomes := make(chan outcome, len(keys))

	var wg sync.WaitGroup
	for _, key := range keys {
		key := key
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			_, _, err := repo.MakeTransaction(ctx, transferParams(key, 5000))
			outcomes <- outcome{err: err}
		}()
	}

	close(start)
	wg.Wait()
	close(outcomes)

	successes, insufficient := 0, 0
	for result := range outcomes {
		switch result.err {
		case nil:
			successes++
		case repository.ErrInsufficientFunds:
			insufficient++
		default:
			t.Errorf("unexpected concurrent transfer error: %v", result.err)
		}
	}

	if successes != 1 || insufficient != 1 {
		t.Errorf("concurrent outcomes: successes=%d insufficient=%d; want one success and one insufficient-funds error",
			successes, insufficient)
	}

	if got := readWalletBalance(t, ctx, pool, testSourceWallet); got != 1000 {
		t.Errorf("source balance = %d, want 1000", got)
	}
	if got := readWalletBalance(t, ctx, pool, testDestinationWallet); got != 6000 {
		t.Errorf("destination balance = %d, want 6000", got)
	}

	transactions, postings := readRecordCounts(t, ctx, pool)
	if transactions != 1 || postings != 2 {
		t.Errorf("persisted records: transactions=%d postings=%d; want 1 and 2", transactions, postings)
	}
}
