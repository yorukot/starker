package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/yorukot/stargo/internal/repository"
	"github.com/yorukot/stargo/internal/service/auth"
	"github.com/yorukot/stargo/pkg/encrypt"
	"github.com/yorukot/stargo/pkg/response"
)

// Register godoc
// @Summary Register a new user
// @Description Creates a new user account with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.RegisterRequest true "Registration request"
// @Success 200 {object} string "User registered successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body or email already in use"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /auth/register [post]
func (h *App) Register(w http.ResponseWriter, r *http.Request) {
	var registerRequest auth.RegisterRequest
	err := json.NewDecoder(r.Body).Decode(&registerRequest)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	if err = auth.RegisterValidate(registerRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		zap.L().Error("Failed to begin transaction", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
		return
	}
	defer tx.Rollback(r.Context())

	// Check if the user already exists
	checkedAccount, err := repository.GetAccountByEmail(r.Context(), tx, registerRequest.Email)
	if err != nil {
		zap.L().Error("Failed to check if user already exists", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to check if user already exists", "FAILED_TO_CHECK_IF_USER_EXISTS")
		return
	}

	if checkedAccount != nil {
		response.RespondWithError(w, http.StatusBadRequest, "This email is already in use", "EMAIL_ALREADY_IN_USE")
		return
	}

	// Generate the full user object
	user, account, err := auth.GenerateUser(registerRequest)
	if err != nil {
		zap.L().Error("Failed to generate user", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to generate user", "FAILED_TO_GENERATE_USER")
		return
	}

	// Create the user and account
	if err = repository.CreateUserAndAccount(r.Context(), tx, user, account); err != nil {
		zap.L().Error("Failed to create user", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to create user", "FAILED_TO_CREATE_USER")
		return
	}

	// Commit the transaction
	if err = tx.Commit(r.Context()); err != nil {
		zap.L().Error("Failed to commit transaction", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to commit transaction", "FAILED_TO_COMMIT_TRANSACTION")
		return
	}

	response.RespondWithJSON(w, http.StatusOK, "User registered successfully", nil)
}

// Login godoc
// @Summary User login
// @Description Authenticates a user with email and password, returns a refresh token cookie
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.LoginRequest true "Login request"
// @Success 200 {object} string "Login successful"
// @Failure 400 {object} response.ErrorResponse "Invalid request body, user not found, or invalid password"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /auth/login [post]
func (h *App) Login(w http.ResponseWriter, r *http.Request) {
	var loginRequest auth.LoginRequest
	err := json.NewDecoder(r.Body).Decode(&loginRequest)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	if err = auth.LoginValidate(loginRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		zap.L().Error("Failed to begin transaction", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
		return
	}
	defer tx.Rollback(r.Context())

	// Get the user by email
	user, err := repository.GetUserByEmail(r.Context(), tx, loginRequest.Email)
	if err != nil {
		zap.L().Error("Failed to get user by email", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get user by email", "FAILED_TO_GET_USER_BY_EMAIL")
		return
	}

	if user == nil {
		response.RespondWithError(w, http.StatusBadRequest, "User not found", "USER_NOT_FOUND")
		return
	}

	// Check if the password is correct
	match, err := encrypt.ComparePasswordAndHash(loginRequest.Password, *user.PasswordHash)
	if err != nil {
		zap.L().Error("Failed to compare password and hash", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to compare password and hash", "FAILED_TO_COMPARE_PASSWORD_AND_HASH")
		return
	}

	if !match {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid password", "INVALID_PASSWORD")
		return
	}

	refreshToken, err := h.GenerateTokenAndSaveRefreshToken(r.Context(), tx, user.ID, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		zap.L().Error("Failed to generate refresh token", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to generate refresh token", "FAILED_TO_GENERATE_REFRESH_TOKEN")
		return
	}

	cookie := http.Cookie{
		Name:     "refresh_token",
		Path:     "/api/auth/refresh",
		Value:    refreshToken.Token,
		HttpOnly: true,
		Secure:   os.Getenv("APP_ENV") == "production",
		Expires:  refreshToken.CreatedAt.Add(time.Hour * 24 * 30),
		SameSite: http.SameSiteStrictMode,
	}

	http.SetCookie(w, &cookie)

	response.RespondWithJSON(w, http.StatusOK, "Login successful", nil)
}
