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
