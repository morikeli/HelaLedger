package services

import "github.com/morikeli/helapay-api/internal/repository"

type TransactionService struct {
	transactionRepository repository.TransactionRepository
}

func NewTransactionService(txnRepo repository.TransactionRepository) *TransactionService {
	return &TransactionService{transactionRepository: txnRepo}
}

