package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/config"
	"github.com/yorukot/starker/internal/repository"
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

// CheckUserHaveAccessToTeam is the middleware for checking if the user have the right to access team
func CheckUserHaveAccessToTeam(db *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get teamID from URL parameters
			teamID := chi.URLParam(r, "teamID")
			if teamID == "" {
				response.RespondWithError(w, http.StatusBadRequest, "Team ID is required", "TEAM_ID_REQUIRED")
				return
			}

			// Get userID from context (set by AuthRequiredMiddleware)
			userID, ok := r.Context().Value(UserIDKey).(string)
			if !ok || userID == "" {
				response.RespondWithError(w, http.StatusUnauthorized, "User not authenticated", "USER_NOT_AUTHENTICATED")
				return
			}

			// Start database transaction
			tx, err := repository.StartTransaction(db, r.Context())
			if err != nil {
				zap.L().Error("Failed to begin transaction", zap.Error(err))
				response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
				return
			}
			defer repository.DeferRollback(tx, r.Context())

			// Check if user has access to the team
			hasAccess, err := repository.CheckTeamAccess(r.Context(), tx, teamID, userID)
			if err != nil {
				zap.L().Error("Failed to check team access", zap.Error(err))
				response.RespondWithError(w, http.StatusInternalServerError, "Failed to check team access", "FAILED_TO_CHECK_TEAM_ACCESS")
				return
			}

			if !hasAccess {
				response.RespondWithError(w, http.StatusForbidden, "Team access denied", "TEAM_ACCESS_DENIED")
				return
			}

			// Commit transaction
			repository.CommitTransaction(tx, r.Context())

			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

