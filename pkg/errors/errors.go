package errs

import (
	"encoding/json"
	"net/http"
)

// Error is message to marshal
type Error struct {
	Message string `json:"message,omitempty"`
	Details string `json:"details,omitempty"`
	Code    int    `json:"code,omitempty"`
}

// New creates a new error
func New(message string, err error, code int) *Error {
	var details string
	if err != nil {
		details = err.Error()
	}
	return &Error{message, details, code}
}

// Write writes error  passed to the writer using json marshaler
func Write(w http.ResponseWriter, err *Error) error {
	w.WriteHeader(err.Code)
	return json.NewEncoder(w).Encode(err)
}
