package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"github.com/yorukot/starker/internal/config"
	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/internal/service/authsvc"
	"github.com/yorukot/starker/pkg/response"
)

// +----------------------------------------------+
// | OAuth Entry                                  |
// +----------------------------------------------+

// OAuthEntry godoc
// @Summary Initiate OAuth flow
// @Description Redirects user to OAuth provider for authentication
// @Tags oauth
// @Param provider path string true "OAuth provider (e.g., google, github)"
// @Param next query string false "Redirect URL after successful OAuth linking"
// @Success 307 {string} string "Redirect to OAuth provider"
// @Failure 400 {object} map[string]interface{} "Invalid provider or bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/oauth/{provider} [get]
func (h *OAuthHandler) OAuthEntry(w http.ResponseWriter, r *http.Request) {
	// Parse the provider
	provider, err := authsvc.ParseProvider(chi.URLParam(r, "provider"))
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid provider", "INVALID_PROVIDER")
		return
	}

	var userID string
	if userIDValue := r.Context().Value(middleware.UserIDKey); userIDValue != nil {
		userID = userIDValue.(string)
	}

	expiresAt := time.Now().Add(time.Duration(config.Env().OAuthStateExpiresAt) * time.Second)

	next := r.URL.Query().Get("next")
	if next == "" {
		next = "/"
	}

	oauthStateJwt, oauthState, err := authsvc.OAuthGenerateStateWithPayload(next, expiresAt, userID)
	if err != nil {
		zap.L().Error("Failed to generate oauth state", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to generate oauth state", "FAILED_TO_GENERATE_OAUTH_STATE")
		return
	}

	// Get the oauth config
	oauthConfig := h.OAuthConfig.Providers[provider]

	// Generate the auth URL
	authURL := oauthConfig.AuthCodeURL(oauthState, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))

	oauthSessionCookie := authsvc.GenerateOAuthSessionCookie(oauthStateJwt)
	http.SetCookie(w, &oauthSessionCookie)

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// +----------------------------------------------+
// | OAuth Callback                               |
// +----------------------------------------------+

// OAuthCallback godoc
// @Summary OAuth callback handler
// @Description Handles OAuth provider callback, processes authorization code, creates/links user accounts, and issues authentication tokens
// @Tags oauth
// @Accept json
// @Produce json
// @Param provider path string true "OAuth provider (e.g., google, github)"
// @Param code query string true "Authorization code from OAuth provider"
// @Param state query string true "OAuth state parameter for CSRF protection"
// @Success 307 {string} string "Redirect to success URL with authentication cookies set"
// @Failure 400 {object} response.ErrorResponse "Invalid provider, oauth state, or verification failed"
// @Failure 500 {object} response.ErrorResponse "Internal server error during user creation or token generation"
// @Router /auth/oauth/{provider}/callback [get]
func (h *OAuthHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	// Get the oauth state from the query params
	oauthState := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	// Get the oauth session cookie
	oauthSessionCookie, err := r.Cookie(models.CookieNameOAuthSession)
	if err != nil {
		zap.L().Debug("OAuth session cookie not found", zap.Error(err))
		response.RespondWithError(w, http.StatusBadRequest, "OAuth session not found", "OAUTH_SESSION_NOT_FOUND")
		return
	}

	// Parse the provider
	provider, err := authsvc.ParseProvider(chi.URLParam(r, "provider"))
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid provider", "INVALID_PROVIDER")
		return
	}

	// No need to check if the provider is valid because it's checked in the ParseProvider function
	oauthConfig := h.OAuthConfig.Providers[provider]
	// Get the oidc provider
	oidcProvider := h.OAuthConfig.OIDCProviders[provider]

	// Validate the oauth state
	valid, payload, err := authsvc.OAuthValidateStateWithPayload(oauthSessionCookie.Value)
	if err != nil || !valid || oauthState != payload.State {
		zap.L().Warn("OAuth state validation failed",
			zap.String("ip", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
			zap.String("provider", string(provider)),
			zap.String("oauth_state", oauthState),
			zap.String("payload_state", payload.State))
		response.RespondWithError(w, http.StatusBadRequest, "Invalid oauth state", "INVALID_OAUTH_STATE")
		return
	}

	// Get the user ID from the session cookie
	var userID string
	var accountID string
	if payload.Subject != "" {
		userID = payload.Subject
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	// Exchange the code for a token
	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Failed to exchange code", "FAILED_TO_EXCHANGE_CODE")
		return
	}

	// Get the raw ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		zap.L().Error("Failed to get id token")
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get id token", "FAILED_TO_GET_ID_TOKEN")
		return
	}

	// Verify the token
	userInfo, err := authsvc.OAuthVerifyTokenAndGetUserInfo(r.Context(), rawIDToken, token, oidcProvider, oauthConfig)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Failed to verify token", "FAILED_TO_VERIFY_TOKEN")
		return
	}

	// Begin the transaction
	tx, err := repository.StartTransaction(h.DB, r.Context())
	if err != nil {
		zap.L().Error("Failed to begin transaction", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
		return
	}
	defer repository.DeferRollback(tx, r.Context())

	// Get the account and user by the provider and user ID for checking if the user is already linked/registered
	account, user, err := repository.GetAccountWithUserByProviderUserID(r.Context(), tx, provider, userInfo.Subject)
	if err != nil {
		zap.L().Error("Failed to get account", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get account", "FAILED_TO_GET_ACCOUNT")
		return
	}

	// If the account is not found and the userID is not nil, it means the user is already registered
	// so we need to link the account to the user
	if user == nil && userID != "" {
		// Link the account to the user
		account, err := authsvc.GenerateUserAccountFromOAuthUserInfo(userInfo, provider, userID)
		if err != nil {
			zap.L().Error("Failed to link account", zap.Error(err))
			response.RespondWithError(w, http.StatusInternalServerError, "Failed to generate user and account", "FAILED_TO_GENERATE_USER_AND_ACCOUNT")
			return
		}

		accountID = account.ID

		// Create the account
		if err = repository.CreateAccount(r.Context(), tx, account); err != nil {
			zap.L().Error("Failed to create account", zap.Error(err))
			response.RespondWithError(w, http.StatusInternalServerError, "Failed to create user and account", "FAILED_TO_CREATE_USER_AND_ACCOUNT")
			return
		}
		zap.L().Info("OAuth new user registered",
			zap.String("provider", string(provider)),
			zap.String("user_id", userID),
			zap.String("ip", r.RemoteAddr))
	} else if account == nil && userID == "" {
		// Generate the full user object
		user, account, err := authsvc.GenerateUserFromOAuthUserInfo(userInfo, provider)
		if err != nil {
			zap.L().Error("Failed to generate user", zap.Error(err))
			response.RespondWithError(w, http.StatusInternalServerError, "Failed to generate user and account", "FAILED_TO_GENERATE_USER_AND_ACCOUNT")
			return
		}

		// Create the user and account
		if err = repository.CreateUserAndAccount(r.Context(), tx, user, account); err != nil {
			zap.L().Error("Failed to create user", zap.Error(err))
			response.RespondWithError(w, http.StatusInternalServerError, "Failed to create user and account", "FAILED_TO_CREATE_USER_AND_ACCOUNT")
			return
		}

		accountID = account.ID

		// Set the user ID to the user ID
		userID = user.ID
		zap.L().Info("OAuth link account successful",
			zap.String("provider", string(provider)),
			zap.String("user_id", userID),
			zap.String("ip", r.RemoteAddr))
	} else {
		accountID = account.ID
		userID = user.ID
		zap.L().Info("OAuth login successful",
			zap.String("provider", string(provider)),
			zap.String("user_id", userID),
			zap.String("ip", r.RemoteAddr))
	}

	// If the user ID is empty, it means something went wrong (it should not happen)
	if userID == "" {
		zap.L().Error("User ID is nil", zap.Any("user", user))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to create user and account", "FAILED_TO_CREATE_USER_AND_ACCOUNT")
		return
	}

	// Create the oauth token
	err = repository.CreateOAuthToken(r.Context(), tx, models.OAuthToken{
		AccountID:    accountID,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
		TokenType:    token.TokenType,
		Provider:     provider,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	if err != nil {
		zap.L().Error("Failed to create oauth token", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to create oauth token", "FAILED_TO_CREATE_OAUTH_TOKEN")
		return
	}

	// Generate the refresh token
	refreshToken, err := GenerateTokenAndSaveRefreshToken(r.Context(), tx, userID, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		zap.L().Error("Failed to create refresh token and access token", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to create refresh token and access token", "FAILED_TO_CREATE_REFRESH_TOKEN_AND_ACCESS_TOKEN")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Generate the refresh token cookie
	refreshTokenCookie := authsvc.GenerateRefreshTokenCookie(refreshToken)
	http.SetCookie(w, &refreshTokenCookie)

	// Redirect to the redirect URI
	http.Redirect(w, r, payload.RedirectURI, http.StatusTemporaryRedirect)
}
