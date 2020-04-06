package middleware

import (
	"net/http"
)

// SetJSONCtype sets the content type to be application/json
func SetJSONCtype(f http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		f.ServeHTTP(w, r)
	})
}
