package responses

import (
	// "encoding/json"
	"encoding/json/v2"
	"net/http"
)

type SuccessResponse struct {
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

func Success(w http.ResponseWriter, code int, message string, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.MarshalWrite(w, SuccessResponse{Message: message, Data: data})
}

func Error(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.MarshalWrite(w, ErrorResponse{Message: message})
}
