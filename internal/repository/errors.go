package repository

import "errors"

var (
	ErrIdempotencyConflict = errors.New("Idempotency key was used with a different payload!")
	ErrRequestInProgress   = errors.New("Request with this idempotency key is still processing!")
	ErrWalletNotFound      = errors.New("Wallet not found!")
	ErrInsufficientFunds   = errors.New("Insufficient funds!")
	ErrCurrencyMismatch    = errors.New("Wallet currency does not match transfer currency!")
	ErrInvalidTransfer     = errors.New("Source and destination wallets must be different!")
	ErrWalletInactive      = errors.New("Wallet is inactive!")
	ErrBalanceOverflow     = errors.New("Destination balance would overflow!")
	ErrInvalidCursor       = errors.New("Invalid pagination cursor!")
)
