package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/ksuid"
	"golang.org/x/oauth2"

	"github.com/yorukot/stargo/internal/models"
	"github.com/yorukot/stargo/internal/repository"
	"github.com/yorukot/stargo/pkg/encrypt"
)

// Check if the provider is valid
func ParseProvider(s string) (models.Provider, error) {
	switch s {
	case string(models.ProviderGoogle):
		return models.ProviderGoogle, nil
	default:
		return "", fmt.Errorf("unknown provider: %s", s)
	}
}

// OAuthGenerateStateWithPayload generates a state for the OAuth flow
func OAuthGenerateStateWithPayload(from string) (string, error) {
	oauthState := uuid.New().String()

	payload := jwt.MapClaims{
		"state": oauthState,
		"from":  from,
	}

	key := []byte(os.Getenv("JWT_SECRET"))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return tokenString, nil
}

// OAuthValidateStateWithPayload validates the state for the OAuth flow
func OAuthValidateStateWithPayload(state string) (jwt.MapClaims, error) {
	key := os.Getenv("JWT_SECRET")
	token, err := jwt.Parse(state, func(token *jwt.Token) (interface{}, error) {
		return []byte(key), nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("failed to get claims from JWT")
	}

	return claims, nil
}

// OauthVerifyToken verifies the token for the OAuth flow
func OAuthVerifyTokenAndGetUserInfo(ctx context.Context, rawIDToken string, token *oauth2.Token, oidcProvider *oidc.Provider, oauthConfig *oauth2.Config) (*oidc.UserInfo, error) {

	// Create verifier with client ID for audience validation
	verifier := oidcProvider.Verifier(&oidc.Config{ClientID: oauthConfig.ClientID})

	// Verify the ID token
	verifiedToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	// Extract claims from verified token
	var tokenClaims map[string]interface{}
	if err := verifiedToken.Claims(&tokenClaims); err != nil {
		return nil, fmt.Errorf("failed to extract claims: %w", err)
	}

	userInfo, err := oidcProvider.UserInfo(ctx, oauth2.StaticTokenSource(token))
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	return userInfo, nil
}

func OAuthRegisterOrLoginUser(ctx context.Context, pool *pgxpool.Pool, userInfo *oidc.UserInfo, provider models.Provider) (*models.Account, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check if the account already exists
	account, err := repository.GetAccountByProviderAndProviderUserID(ctx, tx, provider, userInfo.Subject)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if account != nil {
		return account, nil
	}

	// Get the picture from the user info
	var picture *string
	var claims struct {
		Picture string `json:"picture"`
	}
	if err := userInfo.Claims(&claims); err == nil && claims.Picture != "" {
		picture = &claims.Picture
	}

	// Create a new user variable
	newUser := models.User{
		ID:        ksuid.New().String(),
		Password:  nil,
		Avatar:    picture,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create a new account variable
	newAccount := models.Account{
		ID:             ksuid.New().String(),
		Provider:       provider,
		ProviderUserID: userInfo.Subject,
		UserID:         newUser.ID,
		Email:          userInfo.Email,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Create the user
	if err := repository.CreateUser(ctx, tx, newUser); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create the account
	if err := repository.CreateAccount(ctx, tx, newAccount); err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &newAccount, nil
}

// OAuthCreateRefreshToken creates a new refresh token for the user
func OAuthCreateRefreshTokenAndAccessToken(ctx context.Context, r *http.Request, pool *pgxpool.Pool, userID string) (*models.RefreshToken, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	token, err := encrypt.GenerateSecureRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate secure refresh token: %w", err)
	}

	// Extract IP address without port from RemoteAddr
	clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If SplitHostPort fails, use RemoteAddr as-is (might be just an IP without port)
		clientIP = r.RemoteAddr
	}

	refreshToken := models.RefreshToken{
		ID:        ksuid.New().String(),
		UserID:    userID,
		Token:     token,
		UserAgent: r.Header.Get("User-Agent"),
		IP:        clientIP,
		UsedAt:    nil,
		CreatedAt: time.Now(),
	}

	if err := repository.CreateRefreshToken(ctx, tx, refreshToken); err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &refreshToken, nil
}

// GenrateAccessToken generates a new access token for the user
func GenrateAccessToken(ctx context.Context, userID string, refreshTokenID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(15 * time.Minute).Unix(),
		"iat": time.Now().Unix(),
		"iss": "YOUR_SERVICE_NAME",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
