package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/morikeli/helapay-api/internal/models"
	"github.com/morikeli/helapay-api/internal/repository"
	"github.com/morikeli/helapay-api/internal/services"
)

type insufficientFundsRepository struct{}

func (insufficientFundsRepository) MakeTransaction(context.Context, models.TransactionParams) (*models.TransactionResults, bool, error) {
	return nil, false, repository.ErrInsufficientFunds
}
func (insufficientFundsRepository) GetWalletBalance(context.Context, string) (*models.WalletBalance, error) {
	return nil, errors.New("not implemented")
}
func (insufficientFundsRepository) ListWalletTransactions(context.Context, string, int, string) (*models.TransactionHistory, error) {
	return nil, errors.New("not implemented")
}
func (insufficientFundsRepository) CheckLedgerBalance(context.Context) (bool, error) {
	return false, errors.New("not implemented")
}

func TestCreateTransactionHandler_InsufficientFundsReturnsBadRequest(t *testing.T) {
	handler := NewTransactionHandler(services.NewTransactionService(insufficientFundsRepository{}))
	body := `{"source_wallet_id":"11111111-1111-4111-8111-111111111111","destination_wallet_id":"22222222-2222-4222-8222-222222222222","amount":10001,"currency":"KES"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/transfers", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	response := httptest.NewRecorder()

	handler.CreateTransactionHandler(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d; body: %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), repository.ErrInsufficientFunds.Error()) {
		t.Errorf("response body %q does not contain insufficient-funds message", response.Body.String())
	}
}
