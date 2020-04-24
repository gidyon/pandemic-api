package middleware

// import (
// 	"encoding/json"
// 	"github.com/gidyon/insurance-app/pkg/errs"
// 	"github.com/gidyon/insurance-app/pkg/token"
// 	"net/http"
// 	"strings"
// )

// func parse(w http.ResponseWriter, r *http.Request) (*token.Claims, bool) {
// 	bearerToken := r.Header.Get("authorization")
// 	if bearerToken == "" {
// 		json.NewEncoder(w).Encode(errs.Error{
// 			Message: "missing authorization header",
// 			Details: "authorization header is missing in request; authentication fails",
// 			Code:    http.StatusBadRequest,
// 		})
// 		return nil, false
// 	}
// 	vals := strings.Split(bearerToken, " ")
// 	if len(vals) != 2 {
// 		json.NewEncoder(w).Encode(errs.Error{
// 			Message: "bad format for bearer token",
// 			Details: "authorization header should have payload format: Bearer ei01idj902-f0eii...",
// 			Code:    http.StatusBadRequest,
// 		})
// 		return nil, false
// 	}
// 	claims, err := token.ParseToken(vals[1])
// 	if err != nil {
// 		json.NewEncoder(w).Encode(errs.Error{
// 			Message: "failed to parse token",
// 			Details: err.Error(),
// 			Code:    http.StatusBadRequest,
// 		})
// 		return nil, false
// 	}

// 	return claims, true
// }

// // AuthenticateAdmin authenticates an admin using token in authorization header
// func AuthenticateAdmin(f http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		claims, ok := parse(w, r)
// 		if !ok {
// 			return
// 		}
// 		if !claims.IsAdmin {
// 			json.NewEncoder(w).Encode(errs.Error{
// 				Message: "admin priviledges required",
// 				Details: "",
// 				Code:    http.StatusBadRequest,
// 			})
// 			return
// 		}
// 		f.ServeHTTP(w, r)
// 	})
// }
