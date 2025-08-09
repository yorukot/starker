package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/yorukot/starker/internal/config"
	"github.com/yorukot/starker/pkg/encrypt"
	"github.com/yorukot/starker/pkg/response"
)

// authMiddlewareLogic is the logic for the auth middleware
func authMiddlewareLogic(w http.ResponseWriter, token string) (bool, *encrypt.AccessTokenClaims, error) {
	token = strings.TrimPrefix(token, "Bearer ")

	JWTSecret := encrypt.JWTSecret{
		Secret: config.Env().JWTSecretKey,
	}

	valid, claims, err := JWTSecret.ValidateAccessTokenAndGetClaims(token)
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "Internal server error", "INTERNAL_SERVER_ERROR")
		return false, nil, err
	}

	if !valid {
		response.RespondWithError(w, http.StatusUnauthorized, "Invalid token", "INVALID_TOKEN")
		return false, nil, nil
	}

	return true, &claims, nil
}

// AuthRequiredMiddleware is the middleware for the auth required
func AuthRequiredMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			response.RespondWithError(w, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
			return
		}

		valid, claims, err := authMiddlewareLogic(w, token)
		if err != nil || !valid {
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.Subject)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AuthOptionalMiddleware is the middleware for the auth optional
func AuthOptionalMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}

		valid, claims, err := authMiddlewareLogic(w, token)
		if err != nil || !valid {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.Subject)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
