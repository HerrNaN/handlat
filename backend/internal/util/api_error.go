package util

import (
	"encoding/json"
	"net/http"
)

// WriteAPIError sends a stable code and optional numeric message parameters.
func WriteAPIError(w http.ResponseWriter, status int, code string, params map[string]int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Code   string         `json:"code"`
		Params map[string]int `json:"params,omitempty"`
	}{Code: code, Params: params})
}
