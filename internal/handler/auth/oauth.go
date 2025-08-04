package auth

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"github.com/yorukot/stargo/internal/config"
	"github.com/yorukot/stargo/internal/models"
	"github.com/yorukot/stargo/internal/service"
	"github.com/yorukot/stargo/pkg/utils"
)

// OAuthHandler is the handler for the OAuth providers
type OAuthHandler struct {
	DB          *pgxpool.Pool
	OAuthConfig *config.OAuthConfig
}

// TODO: We also can get the user's cookie and check if the user is already logged in to know user want to link or login/register

// OauthEntry handles the OAuth entry flow
// @Summary Initiate OAuth flow
// @Description Redirects user to OAuth provider for authentication
// @Tags oauth
// @Param provider path string true "OAuth provider (e.g., google, github)"
// @Param from query string false "Redirect URL after successful authentication"
// @Success 307 {string} string "Redirect to OAuth provider"
// @Failure 400 {object} map[string]interface{} "Invalid provider or bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/oauth/{provider} [get]
func (h *OAuthHandler) OauthEntry(w http.ResponseWriter, r *http.Request) {
	provider, err := service.ParseProvider(chi.URLParam(r, "provider"))
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error(), "UNKNOWN_PROVIDER")
		return
	}

	oauthState, err := service.OAuthGenerateStateWithPayload(r.URL.Query().Get("from"))
	if err != nil {
		zap.L().Error("failed to generate oauth state", zap.Error(err))
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error(), "INTERNAL_SERVER_ERROR")
		return
	}

	oauthConfig, ok := h.OAuthConfig.Providers[provider]
	if !ok {
		utils.RespondWithError(w, http.StatusBadRequest, "provider not found", "UNKNOWN_PROVIDER")
		return
	}

	authURL := oauthConfig.AuthCodeURL(oauthState, oauth2.AccessTypeOffline)

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// OauthCallback handles the OAuth callback flow
// @Summary Handle OAuth callback
// @Description Processes OAuth callback from provider and authenticates user
// @Tags oauth
// @Param provider path string true "OAuth provider (e.g., google, github)"
// @Param code query string true "Authorization code from OAuth provider"
// @Param state query string true "State parameter for CSRF protection"
// @Success 200 {object} map[string]interface{} "Successful authentication with refresh token"
// @Failure 400 {object} map[string]interface{} "Invalid provider, code, or state"
// @Failure 401 {object} map[string]interface{} "Invalid or expired token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/oauth/{provider}/callback [get]
func (h *OAuthHandler) OauthCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stateParam := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	provider, err := service.ParseProvider(chi.URLParam(r, "provider"))
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error(), "UNKNOWN_PROVIDER")
		return
	}

	// Get the OAuth config for the provider
	oauthConfig, ok := h.OAuthConfig.Providers[provider]
	if !ok {
		utils.RespondWithError(w, http.StatusBadRequest, "provider not found", "UNKNOWN_PROVIDER")
		return
	}

	// Validate the state, if the user need to link we can use this to know who is the user
	_, err = service.OAuthValidateStateWithPayload(stateParam)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid state", "INVALID_STATE")
		return
	}

	// TODO: save the refresh token and access token to the database
	// Exchange the code for a token
	token, err := oauthConfig.Exchange(r.Context(), code)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "failed to exchange code", "INVALID_GRANT")
		return
	}

	// Get the ID token for the OIDC verification from the exchanged token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		zap.L().Error("failed to get id token", zap.Error(err))
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to get id token", "INTERNAL_SERVER_ERROR")
		return
	}

	// Get the OIDC provider for the oidc verification
	oidcProvider, ok := h.OAuthConfig.OIDCProviders[provider]
	if !ok {
		zap.L().Error("OIDC provider not configured")
		utils.RespondWithError(w, http.StatusInternalServerError, "OIDC provider not configured", "INTERNAL_SERVER_ERROR")
		return
	}

	// Verify the ID token for the OIDC provider
	userInfo, err := service.OAuthVerifyTokenAndGetUserInfo(ctx, rawIDToken, token, oidcProvider, oauthConfig)
	if err != nil {
		zap.L().Error("failed to verify ID token", zap.Error(err))
		utils.RespondWithError(w, http.StatusUnauthorized, "failed to verify ID token", "INVALID_TOKEN")
		return
	}

	account, err := service.OAuthRegisterOrLoginUser(ctx, h.DB, userInfo, models.Provider(provider))
	if err != nil {
		zap.L().Error("failed to register or login user", zap.Error(err))
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to register or login user", "INTERNAL_SERVER_ERROR")
		return
	}

	// Set the cookie for the user
	refreshToken, err := service.OAuthCreateRefreshTokenAndAccessToken(ctx, r, h.DB, account.UserID)
	if err != nil {
		zap.L().Error("failed to create refresh token", zap.Error(err))
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to create refresh token", "INTERNAL_SERVER_ERROR")
		return
	}

	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken.Token,
		Path:     "/api/auth/refresh",
		HttpOnly: true,
		Secure:   os.Getenv("ENV") == "production",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   60 * 60 * 24 * 365, // 1 year
	}

	http.SetCookie(w, cookie)

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "user registered or logged in successfully",
	})
}
