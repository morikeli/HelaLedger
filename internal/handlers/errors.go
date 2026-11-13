package handlers

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/morikeli/helapay-api/internal/dtos/responses"
	"github.com/morikeli/helapay-api/internal/repository"
)

func handleServiceError(w http.ResponseWriter, err error) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "40P01" || pgErr.Code == "40001" || pgErr.Code == "55P03") {
		w.Header().Set("Retry-After", "1")
		responses.Error(w, http.StatusServiceUnavailable, "temporary database contention; retry with the same idempotency key")
		return
	}
	switch {
	case errors.Is(err, repository.ErrInvalidTransfer), errors.Is(err, repository.ErrInvalidCursor):
		responses.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, repository.ErrInsufficientFunds):
		responses.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, repository.ErrCurrencyMismatch), errors.Is(err, repository.ErrWalletInactive), errors.Is(err, repository.ErrBalanceOverflow):
		responses.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, repository.ErrWalletNotFound):
		responses.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, repository.ErrIdempotencyConflict):
		responses.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, repository.ErrRequestInProgress):
		w.Header().Set("Retry-After", "1")
		responses.Error(w, http.StatusTooEarly, err.Error())
	default:
		responses.Error(w, http.StatusInternalServerError, "an unexpected error occurred")
	}
}
