package auth

import (
	"fmt"
	"net"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/segmentio/ksuid"

	"github.com/yorukot/stargo/internal/models"
	"github.com/yorukot/stargo/pkg/encrypt"
)

// RegisterRequest is the request body for the register endpoint
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=255"`
}

// TODO: write tests for this function
// We separate the validation logic from the handler to make it easier to test
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
		ID:        userID,
		PasswordHash:  &passwordHash,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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

// TODO: write tests for this function
// LoginValidate validates the login request
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
		UserAgent: userAgent,
		IP:        ipStr,
		UsedAt:    nil,
		CreatedAt: time.Now(),
	}, nil
}
