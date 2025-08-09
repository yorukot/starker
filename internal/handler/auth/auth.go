// Package auth provides the auth handler.
package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/yorukot/stargo/internal/config"
	"github.com/yorukot/stargo/internal/repository"
	"github.com/yorukot/stargo/internal/service/authsvc"
	"github.com/yorukot/stargo/pkg/encrypt"
	"github.com/yorukot/stargo/pkg/response"
)

// AuthHandler is the handler for the auth routes
type AuthHandler struct {
	DB *pgxpool.Pool
}

// +----------------------------------------------+
// | Register                                     |
// +----------------------------------------------+

// Register godoc
// @Summary Register a new user
// @Description Creates a new user account with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body authsvc.RegisterRequest true "Registration request"
// @Success 200 {object} string "User registered successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body or email already in use"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	// Decode the request body
	var registerRequest authsvc.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&registerRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	// Validate the request body
	if err := authsvc.RegisterValidate(registerRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
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

	// Get the account by email
	checkedAccount, err := repository.GetAccountByEmail(r.Context(), tx, registerRequest.Email)
	if err != nil {
		zap.L().Error("Failed to check if user already exists", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to check if user already exists", "FAILED_TO_CHECK_IF_USER_EXISTS")
		return
	}

	// If the account is found, return an error
	if checkedAccount != nil {
		response.RespondWithError(w, http.StatusBadRequest, "This email is already in use", "EMAIL_ALREADY_IN_USE")
		return
	}

	// Generate the user and account
	user, account, err := authsvc.GenerateUser(registerRequest)
	if err != nil {
		zap.L().Error("Failed to generate user", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to generate user", "FAILED_TO_GENERATE_USER")
		return
	}

	// Create the user and account in the database
	if err = repository.CreateUserAndAccount(r.Context(), tx, user, account); err != nil {
		zap.L().Error("Failed to create user", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to create user", "FAILED_TO_CREATE_USER")
		return
	}

	// Generate the refresh token
	refreshToken, err := GenerateTokenAndSaveRefreshToken(r.Context(), tx, user.ID, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		zap.L().Error("Failed to generate refresh token", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to generate refresh token", "FAILED_TO_GENERATE_REFRESH_TOKEN")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Generate the refresh token cookie
	refreshTokenCookie := authsvc.GenerateRefreshTokenCookie(refreshToken)
	http.SetCookie(w, &refreshTokenCookie)

	// Respond with the success message
	response.RespondWithJSON(w, http.StatusOK, "User registered successfully", nil)
}

// +----------------------------------------------+
// | Login                                        |
// +----------------------------------------------+

// Login godoc
// @Summary User login
// @Description Authenticates a user with email and password, returns a refresh token cookie
// @Tags auth
// @Accept json
// @Produce json
// @Param request body authsvc.LoginRequest true "Login request"
// @Success 200 {object} string "Login successful"
// @Failure 400 {object} response.ErrorResponse "Invalid request body, user not found, or invalid password"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Decode the request body
	var loginRequest authsvc.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	// Validate the request body
	if err := authsvc.LoginValidate(loginRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
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

	// Get the user by email
	user, err := repository.GetUserByEmail(r.Context(), tx, loginRequest.Email)
	if err != nil {
		zap.L().Error("Failed to get user by email", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get user by email", "FAILED_TO_GET_USER_BY_EMAIL")
		return
	}

	// TODO: Need to change this
	// If the user is not found, return an error
	if user == nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid credentials", "INVALID_CREDENTIALS")
		return
	}

	// Compare the password and hash
	match, err := encrypt.ComparePasswordAndHash(loginRequest.Password, *user.PasswordHash)
	if err != nil {
		zap.L().Error("Failed to compare password and hash", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to compare password and hash", "FAILED_TO_COMPARE_PASSWORD_AND_HASH")
		return
	}

	// If the password is not correct, return an error
	if !match {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid credentials", "INVALID_CREDENTIALS")
		return
	}

	// Generate the refresh token
	refreshToken, err := GenerateTokenAndSaveRefreshToken(r.Context(), tx, user.ID, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		zap.L().Error("Failed to generate refresh token", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to generate refresh token", "FAILED_TO_GENERATE_REFRESH_TOKEN")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Generate the refresh token cookie
	refreshTokenCookie := authsvc.GenerateRefreshTokenCookie(refreshToken)
	http.SetCookie(w, &refreshTokenCookie)

	response.RespondWithJSON(w, http.StatusOK, "Login successful", nil)
}

// +----------------------------------------------+
// | Refresh Token                                |
// +----------------------------------------------+

// RefreshToken godoc
// @Summary Refresh token
// @Description Refreshes the access token and returns a new refresh token cookie
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} string "Access token generated successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body, refresh token not found, or refresh token already used"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	userRefreshToken, err := r.Cookie("refresh_token")
	if err != nil {
		response.RespondWithError(w, http.StatusUnauthorized, "Refresh token not found", "REFRESH_TOKEN_NOT_FOUND")
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

	// Get the refresh token by token
	checkedRefreshToken, err := repository.GetRefreshTokenByToken(r.Context(), tx, userRefreshToken.Value)
	if err != nil {
		zap.L().Error("Failed to get refresh token by token", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get refresh token by token", "FAILED_TO_GET_REFRESH_TOKEN_BY_TOKEN")
		return
	}

	// If the refresh token is not found, return an error
	if checkedRefreshToken == nil {
		response.RespondWithError(w, http.StatusUnauthorized, "Refresh token not found", "REFRESH_TOKEN_NOT_FOUND")
		return
	}

	// TODO: Need to tell the user might just been hacked
	if checkedRefreshToken.UsedAt != nil {
		zap.L().Warn("Refresh token already used",
			zap.String("refresh_token_id", checkedRefreshToken.ID),
			zap.String("ip", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
		)
		response.RespondWithError(w, http.StatusUnauthorized, "Refresh token already used", "REFRESH_TOKEN_ALREADY_USED")
		return
	}
	// Update the refresh token used_at
	now := time.Now()
	checkedRefreshToken.UsedAt = &now
	if err = repository.UpdateRefreshTokenUsedAt(r.Context(), tx, *checkedRefreshToken); err != nil {
		zap.L().Error("Failed to update refresh token used_at", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to update refresh token used_at", "FAILED_TO_UPDATE_REFRESH_TOKEN_USED_AT")
		return
	}

	// Generate new refresh token
	newRefreshToken, err := GenerateTokenAndSaveRefreshToken(r.Context(), tx, checkedRefreshToken.UserID, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		zap.L().Error("Failed to generate refresh token", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to generate refresh token", "FAILED_TO_GENERATE_REFRESH_TOKEN")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Generate the refresh token cookie
	refreshTokenCookie := authsvc.GenerateRefreshTokenCookie(newRefreshToken)
	http.SetCookie(w, &refreshTokenCookie)

	// Generate AccessTokenClaims
	accessTokenClaims := encrypt.JWTSecret{
		Secret: config.Env().JWTSecretKey,
	}

	// Generate the access token
	accessToken, err := accessTokenClaims.GenerateAccessToken(config.Env().AppName, checkedRefreshToken.UserID, time.Now().Add(time.Duration(config.Env().AccessTokenExpiresAt)*time.Second))
	if err != nil {
		zap.L().Error("Failed to generate access token", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Internal server error", "INTERNAL_SERVER_ERROR")
		return
	}

	response.RespondWithJSON(w, 200, "Access token generated successfully", map[string]string{
		"access_token": accessToken,
	})
}
