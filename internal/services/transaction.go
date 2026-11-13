package services

import (
	"context"

	"github.com/morikeli/helapay-api/internal/models"
	"github.com/morikeli/helapay-api/internal/repository"
)

type TransactionService struct {
	transactionRepository repository.TransactionRepository
}

func NewTransactionService(txnRepo repository.TransactionRepository) *TransactionService {
	return &TransactionService{transactionRepository: txnRepo}
}

// transact money to users
func (s *TransactionService) TransactFunds(ctx context.Context, params models.TransactionParams) (*models.TransactionResults, bool, error) {
	return s.transactionRepository.MakeTransaction(ctx, params)
}

// fetch wallet balance and recent transaction
func (s *TransactionService) WalletBalance(ctx context.Context, walletID string) (*models.WalletBalance, error) {
	return s.transactionRepository.GetWalletBalance(ctx, walletID)
}

// fetch paginated transaction history
func (s *TransactionService) TransactionHistory(ctx context.Context, walletID string, limit int, cursor string) (*models.TransactionHistory, error) {
	return s.transactionRepository.ListWalletTransactions(ctx, walletID, limit, cursor)
}

// Verify ledger consistency
func (s *TransactionService) IsLedgerBalanced(ctx context.Context) (bool, error) {
	return s.transactionRepository.CheckLedgerBalance(ctx)
}