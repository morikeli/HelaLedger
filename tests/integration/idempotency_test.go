package integration

import (
	"context"
	"testing"

	"github.com/morikeli/helapay-api/internal/repository"
)

func TestMakeTransaction_ReplayedIdempotencyKeyReturnsCachedResponse(t *testing.T) {
	repo, pool := newIntegrationRepository(t)
	ctx := context.Background()

	params := transferParams("eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee", 2500)

	first, replayed, err := repo.MakeTransaction(ctx, params)
	if err != nil {
		t.Fatalf("first MakeTransaction() error = %v", err)
	}
	if replayed {
		t.Fatal("first request unexpectedly marked as replay")
	}

	second, replayed, err := repo.MakeTransaction(ctx, params)
	if err != nil {
		t.Fatalf("replayed MakeTransaction() error = %v", err)
	}
	if !replayed {
		t.Fatal("second request was not identified as a replay")
	}
	if second == nil {
		t.Fatal("replayed response is nil")
	}
	if second.TransactionID != first.TransactionID ||
		second.ReferenceID != first.ReferenceID ||
		second.HTTPStatus != first.HTTPStatus {
		t.Errorf("cached response differs from original: first=%+v second=%+v", first, second)
	}

	if got := readWalletBalance(t, ctx, pool, testSourceWallet); got != 7500 {
		t.Errorf("source balance after replay = %d, want 7500", got)
	}

	transactions, postings := readRecordCounts(t, ctx, pool)
	if transactions != 1 || postings != 2 {
		t.Errorf("replay created duplicate records: transactions=%d postings=%d; want 1 and 2",
			transactions, postings)
	}

	changedPayload := params
	changedPayload.Amount++
	if _, _, err := repo.MakeTransaction(ctx, changedPayload); err != repository.ErrIdempotencyConflict {
		t.Errorf("same key with changed payload error = %v, want ErrIdempotencyConflict", err)
	}
}
