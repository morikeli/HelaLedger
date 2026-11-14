package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	chiMW "github.com/go-chi/chi/v5/middleware"
	"github.com/morikeli/helapay-api/internal/dtos/requests"
	"github.com/morikeli/helapay-api/internal/dtos/responses"
	"github.com/morikeli/helapay-api/internal/models"
	"github.com/morikeli/helapay-api/internal/pkg/payloadutil"
	"github.com/morikeli/helapay-api/internal/pkg/validatorutil"
	"github.com/morikeli/helapay-api/internal/services"
)

type TransactionHandler struct {
	service *services.TransactionService
}

func NewTransactionHandler(service *services.TransactionService) *TransactionHandler {
	return &TransactionHandler{service: service}
}

func (h *TransactionHandler) CreateTransactionHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// validate content type
	if err := validatorutil.ValidateJSONContentType(r); err != nil {
		responses.Error(w, http.StatusUnsupportedMediaType, err.Error())
		return
	}

	// limit request body size
	payloadutil.LimitJSONBody(w, r)

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		responses.Error(w, http.StatusBadRequest, "Missing Idempotency-Key header!")
		return
	}

	// decode request using strict JSON
	var req requests.TransactionRequestDTO
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		if payloadutil.IsRequestBodyTooLarge(err) {
			responses.Error(w, http.StatusRequestEntityTooLarge, "Request body too large!")
			return
		}

		responses.Error(w, http.StatusBadRequest, "Invalid JSON request body!")
		return
	}

	// ensure body contains one JSON payload
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		responses.Error(w, http.StatusBadRequest, "Request body must contain a single JSON object!")
		return
	}

	// validate request fields
	if err := validatorutil.ValidateRequest(&req); err != nil {
		responses.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// map request DTO to domain aprameters
	// normalize uuid casing so semantically identical payloads hash indentically
	params := models.TransactionParams{
		IdempotencyKey:      strings.ToLower(idempotencyKey),
		SourceWalletID:      strings.ToLower(req.SourceWalletID),
		DestinationWalletID: strings.ToLower(req.DestinationWalletID),
		Amount:              req.Amount,
		Currency:            req.Currency,
	}

	// execute transfer
	result, isReplay, err := h.service.TransactFunds(ctx, params)
	if err != nil {
		slog.Warn("wallet transfer rejected", "request_id", chiMW.GetReqID(r.Context()), "error", err.Error())
		handleServiceError(w, err)
		return
	}

	response := responses.TransactionResponseDTO{
		Status:        result.Status,
		TransactionID: result.TransactionID,
		ReferenceID:   result.ReferenceID,
		Amount:        result.Amount,
		Currency:      result.Currency,
	}

	slog.Info("wallet transfer completed", "request_id", chiMW.GetReqID(r.Context()), "transaction_id", result.TransactionID, "reference_id", result.ReferenceID, "amount_minor", result.Amount, "currency", result.Currency, "replayed", isReplay)
	statusCode := http.StatusCreated
	if isReplay {
		statusCode = http.StatusOK
	}

	responses.Success(w, statusCode, "Cash transacted successfully!", response)
}

func (h *TransactionHandler) GetWalletBalanceHandler(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "id")
	ctx := r.Context()

	// validate wallet UUID
	if !validatorutil.ValidateUUID(walletID) {
		responses.Error(w, http.StatusBadRequest, "Wallet ID must be a valid UUID!")
		return
	}

	result, err := h.service.WalletBalance(ctx, walletID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	responses.Success(w, http.StatusOK, "Wallet balance retrieved successfully!", result)
}

func (h *TransactionHandler) GetTransactionHistoryHandler(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "id")
	ctx := r.Context()

	// validate wallet UUID
	if !validatorutil.ValidateUUID(walletID) {
		responses.Error(w, http.StatusBadRequest, "Wallet ID must be a valid UUID!")
		return
	}

	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			responses.Error(w, http.StatusBadRequest, "Limit must be between 1 and 100!")
			return
		}
		limit = parsed
	}

	query := r.URL.Query().Get("cursor")
	result, err := h.service.TransactionHistory(ctx, walletID, limit, query)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	responses.Success(w, http.StatusOK, "Wallet transactions retrieved successfully!", result)
}

func (h *TransactionHandler) CheckLedgerBalanceHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	isBalanced, err := h.service.IsLedgerBalanced(ctx)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	// Map domain result to HTTP Response DTO
	response := responses.LedgerStatusResponseDTO{
		IsBalanced: isBalanced,
	}

	responses.Success(w, http.StatusOK, "Ledger balance status retrieved successfully!", response)
}
