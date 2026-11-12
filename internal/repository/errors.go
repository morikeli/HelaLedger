package repository

import "errors"

var (
	ErrIdempotencyConflict = errors.New("idempotency key was used with a different payload")
	ErrRequestInProgress   = errors.New("request with this idempotency key is still processing")
	ErrWalletNotFound      = errors.New("wallet not found")
	ErrInsufficientFunds   = errors.New("insufficient funds")
	ErrCurrencyMismatch    = errors.New("wallet currency does not match transfer currency")
	ErrInvalidTransfer     = errors.New("source and destination wallets must be different")
	ErrWalletInactive      = errors.New("wallet is not active")
	ErrBalanceOverflow     = errors.New("destination balance would overflow")
	ErrInvalidCursor       = errors.New("invalid pagination cursor")
)