package authsvc

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/segmentio/ksuid"

	"github.com/yorukot/starker/internal/config"
	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/pkg/encrypt"
)

// RegisterRequest is the request body for the register endpoint
type RegisterRequest struct {
	Email       string `json:"email" validate:"required,email,max=255"`
	Password    string `json:"password" validate:"required,min=8,max=255"`
	DisplayName string `json:"display_name" validate:"required,min=3,max=255"`
}

// RegisterValidate validate the register request
func RegisterValidate(registerRequest RegisterRequest) error {
	return validator.New().Struct(registerRequest)
}

// GenerateUser generate a user and account for the register request
func GenerateUser(registerRequest RegisterRequest) (models.User, models.Account, error) {
	userID := ksuid.New().String()

	// hash the password
	passwordHash, err := encrypt.CreateArgon2idHash(registerRequest.Password)
	if err != nil {
		return models.User{}, models.Account{}, fmt.Errorf("failed to hash password: %w", err)
	}

	// create the user
	user := models.User{
		ID:           userID,
		PasswordHash: &passwordHash,
		DisplayName:  registerRequest.DisplayName,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Create the account
	account := models.Account{
		ID:             ksuid.New().String(),
		Provider:       models.ProviderEmail,
		ProviderUserID: userID,
		UserID:         userID,
		Email:          registerRequest.Email,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return user, account, nil
}

// LoginRequest is the request body for the login endpoint
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=255"`
}

// LoginValidate validate the login request
func LoginValidate(loginRequest LoginRequest) error {
	return validator.New().Struct(loginRequest)
}

// GenerateRefreshToken generates a refresh token for the user
func GenerateRefreshToken(userID string, userAgent string, ip string) (models.RefreshToken, error) {
	ipStr, _, err := net.SplitHostPort(ip)
	if err != nil {
		return models.RefreshToken{}, fmt.Errorf("failed to split host port: %w", err)
	}

	refreshToken, err := encrypt.GenerateSecureRefreshToken()
	if err != nil {
		return models.RefreshToken{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return models.RefreshToken{
		ID:        ksuid.New().String(),
		UserID:    userID,
		Token:     refreshToken,
		UserAgent: &userAgent,
		IP:        &ipStr,
		UsedAt:    nil,
		CreatedAt: time.Now(),
	}, nil
}

// GenerateRefreshTokenCookie generates a refresh token cookie
func GenerateRefreshTokenCookie(refreshToken models.RefreshToken) http.Cookie {
	return http.Cookie{
		Name: models.CookieNameRefreshToken,
		Path: "/api/auth/refresh",
		// TODO: Add config for frontend domain
		// Domain:   "localhost",
		Value:    refreshToken.Token,
		HttpOnly: true,
		Secure:   config.Env().AppEnv == config.AppEnvProd,
		Expires:  refreshToken.CreatedAt.Add(time.Duration(config.Env().RefreshTokenExpiresAt) * time.Second),
		SameSite: http.SameSiteLaxMode,
	}
}
