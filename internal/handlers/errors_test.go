package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/morikeli/helapay-api/internal/repository"
)

func TestHandleServiceError_StatusMapping(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		wantStatus     int
		wantRetryAfter bool
	}{
		{"invalid transfer", repository.ErrInvalidTransfer, http.StatusBadRequest, false},
		{"invalid cursor", repository.ErrInvalidCursor, http.StatusBadRequest, false},
		{"insufficient funds", repository.ErrInsufficientFunds, http.StatusBadRequest, false},
		{"wallet missing", repository.ErrWalletNotFound, http.StatusNotFound, false},
		{"idempotency conflict", repository.ErrIdempotencyConflict, http.StatusConflict, false},
		{"request in progress", repository.ErrRequestInProgress, http.StatusTooEarly, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handleServiceError(w, tt.err)
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if got := w.Header().Get("Retry-After") != ""; got != tt.wantRetryAfter {
				t.Errorf("Retry-After present = %v, want %v", got, tt.wantRetryAfter)
			}
		})
	}
}
