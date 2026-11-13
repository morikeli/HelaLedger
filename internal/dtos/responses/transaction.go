package responses

type TransactionResponseDTO struct {
	Status        string `json:"status"`
	TransactionID string `json:"transaction_id"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
}