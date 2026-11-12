package services

import "github.com/morikeli/helapay-api/internal/repository"

type TransactionService struct {
	transactionRepository repository.TransactionRepository
}

func NewTransactionService(transactionRepo repository.TransactionRepository) *TransactionService {
	return &TransactionService{
		TransactionService: transactionRepo,
	}
}

