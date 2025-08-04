package middleware

import "net/http"

// This can be used to validate the access token
// TODO: Need to finish this function right now it's just a placeholder
func ValidateAccessToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	})
}
