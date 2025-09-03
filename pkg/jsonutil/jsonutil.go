package jsonutil

import (
	"encoding/json"
	"net/http"
)

// JSON writes a JSON response with status code.
func JSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
