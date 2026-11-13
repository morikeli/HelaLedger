package hashutil

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/morikeli/helapay-api/internal/models"
)

// Generate a deterministic SHA-256 cryptographic fingerprint for a transaction
func HashPayload(txnData models.TransactionParams) (string, error) {
	// create a canonical (normalized) anonymous struct containing only the
	// essential fields, mapped with explicit lowercase JSON tags
	canonical := struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
		Amount      int64  `json:"amount"`
		Currency    string `json:"currency"`
	}{txnData.SourceWalletID, txnData.DestinationWalletID, txnData.Amount, txnData.Currency}

	// serialize the normalized struct
	b, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}

	// pass the JSON data through a hash function to produce a
	// fixed-size 32-byte array
	sum := sha256.Sum256(b)

	// convert the raw 32-byte array into a human-readable, 64-character
	// hexadecimal string and return it
	return hex.EncodeToString(sum[:]), nil
}
