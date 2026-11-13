package requests

type TransactionRequestDTO struct {
	SourceWalletID      string `json:"source_wallet_id" validate:"required,uuid"`
	DestinationWalletID string `json:"destination_wallet_id" validate:"required,uuid"`
	Amount              int64  `json:"amount" validate:"required,gt=0"` // Integer cents
	Currency            string `json:"currency" validate:"required,len=3,uppercase"`
}
