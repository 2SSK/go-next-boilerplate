// Package httpx holds the JSON response helpers used outside the generated
// handlers: middleware, the 404 fallback, and transport-level errors.
package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/2SSK/go-next-boilerplate/api"
)

// JSON writes v as a JSON response with the given status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Error writes the contract's error shape: {"error": {"code", "message"}}.
func Error(w http.ResponseWriter, status int, code api.ErrorErrorCode, message string) {
	JSON(w, status, api.NewError(code, message))
}
