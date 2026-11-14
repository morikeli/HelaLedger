package services

import (
	"context"
	"errors"
	"testing"

	"github.com/morikeli/helapay-api/internal/models"
)

type fakeTransactionRepository struct {
	makeResult     *models.TransactionResults
	makeReplay     bool
	makeErr        error
	balance        *models.WalletBalance
	balanceErr     error
	history        *models.TransactionHistory
	historyErr     error
	ledgerBalanced bool
	ledgerErr      error

	gotParams   models.TransactionParams
	gotWalletID string
	gotLimit    int
	gotCursor   string
}

func (f *fakeTransactionRepository) MakeTransaction(_ context.Context, p models.TransactionParams) (*models.TransactionResults, bool, error) {
	f.gotParams = p
	return f.makeResult, f.makeReplay, f.makeErr
}
func (f *fakeTransactionRepository) GetWalletBalance(_ context.Context, id string) (*models.WalletBalance, error) {
	f.gotWalletID = id
	return f.balance, f.balanceErr
}
func (f *fakeTransactionRepository) ListWalletTransactions(_ context.Context, id string, limit int, cursor string) (*models.TransactionHistory, error) {
	f.gotWalletID, f.gotLimit, f.gotCursor = id, limit, cursor
	return f.history, f.historyErr
}
func (f *fakeTransactionRepository) CheckLedgerBalance(context.Context) (bool, error) {
	return f.ledgerBalanced, f.ledgerErr
}

func TestTransactionService_TransactFundsDelegates(t *testing.T) {
	want := &models.TransactionResults{Status: "COMPLETED", TransactionID: "txn-1", Amount: 500, Currency: "KES"}
	repo := &fakeTransactionRepository{makeResult: want, makeReplay: true}
	svc := NewTransactionService(repo)
	params := models.TransactionParams{IdempotencyKey: "key", SourceWalletID: "source", DestinationWalletID: "dest", Amount: 500, Currency: "KES"}
	got, replay, err := svc.TransactFunds(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}
	if got != want || !replay {
		t.Fatalf("got (%v, %v), want (%v, true)", got, replay, want)
	}
	if repo.gotParams != params {
		t.Errorf("repository received params %#v, want %#v", repo.gotParams, params)
	}
}

func TestTransactionService_TransactFundsPropagatesError(t *testing.T) {
	wantErr := errors.New("repository unavailable")
	svc := NewTransactionService(&fakeTransactionRepository{makeErr: wantErr})
	_, _, err := svc.TransactFunds(context.Background(), models.TransactionParams{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestTransactionService_WalletBalanceDelegates(t *testing.T) {
	want := &models.WalletBalance{WalletID: "wallet-1", Balance: 2500, Currency: "KES"}
	repo := &fakeTransactionRepository{balance: want}
	got, err := NewTransactionService(repo).WalletBalance(context.Background(), "wallet-1")
	if err != nil {
		t.Fatal(err)
	}
	if got != want || repo.gotWalletID != "wallet-1" {
		t.Fatalf("unexpected result or wallet ID: %#v, %q", got, repo.gotWalletID)
	}
}

func TestTransactionService_TransactionHistoryDelegates(t *testing.T) {
	want := &models.TransactionHistory{}
	repo := &fakeTransactionRepository{history: want}
	got, err := NewTransactionService(repo).TransactionHistory(context.Background(), "wallet-2", 25, "cursor")
	if err != nil {
		t.Fatal(err)
	}
	if got != want || repo.gotWalletID != "wallet-2" || repo.gotLimit != 25 || repo.gotCursor != "cursor" {
		t.Fatal("history arguments/result were not delegated correctly")
	}
}

func TestTransactionService_IsLedgerBalancedDelegates(t *testing.T) {
	repo := &fakeTransactionRepository{ledgerBalanced: true}
	got, err := NewTransactionService(repo).IsLedgerBalanced(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatal("expected balanced ledger result")
	}
}
