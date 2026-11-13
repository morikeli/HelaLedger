package models

type TransactionParams struct {
	IdempotencyKey      string
	SourceWalletID      string
	DestinationWalletID string
	Amount              int64
	Currency            string
}

type TransactionResults struct {
	Status        string
	TransactionID string
	ReferenceID   string
	Amount        int64
	Currency      string
}

type TransactionHistory struct {
	Items      []PostingItem
	NextCursor string      
}