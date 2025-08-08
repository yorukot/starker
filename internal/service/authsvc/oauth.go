package authsvc

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/segmentio/ksuid"
	"golang.org/x/oauth2"

	"github.com/yorukot/stargo/internal/config"
	"github.com/yorukot/stargo/internal/models"
	"github.com/yorukot/stargo/pkg/encrypt"
)

// ParseProvider parse the provider from the request
func ParseProvider(provider string) (models.Provider, error) {
	switch provider {
	case string(models.ProviderGoogle):
		return models.ProviderGoogle, nil
	default:
		return "", fmt.Errorf("invalid provider: %s", provider)
	}
}

// OAuthGenerateStateWithPayload generate the oauth state with the payload
func OAuthGenerateStateWithPayload(redirectURI string, expiresAt time.Time, userID string) (string, error) {
	OAuthState, err := encrypt.GenerateRandomString(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate random string: %w", err)
	}

	secret := encrypt.JWTSecret{
		Secret: config.Env().JWTSecretKey,
	}

	tokenString, err := secret.GenerateOAuthState(OAuthState, redirectURI, expiresAt, userID)
	if err != nil {
		return "", fmt.Errorf("failed to generate oauth state: %w", err)
	}



	return tokenString, nil
}

// OAuthValidateStateWithPayload validate the oauth state with the payload
func OAuthValidateStateWithPayload(oauthState string) (bool, encrypt.OAuthStateClaims, error) {
	secret := encrypt.JWTSecret{
		Secret: config.Env().JWTSecretKey,
	}

	valid, payload, err := secret.ValidateOAuthStateAndGetClaims(oauthState)
	if err != nil {
		return false, encrypt.OAuthStateClaims{}, fmt.Errorf("failed to validate oauth state: %w", err)
	}

	if payload.ExpiresAt < time.Now().Unix() {
		return false, encrypt.OAuthStateClaims{}, fmt.Errorf("oauth state expired")
	}

	return valid, payload, nil
}

// OAuthVerifyTokenAndGetUserInfo verifies the token for the OAuth flow
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

// GenerateUserFromOAuthUserInfo generate the user and account from the oauth user info
func GenerateUserFromOAuthUserInfo(userInfo *oidc.UserInfo, provider models.Provider) (models.User, models.Account, error) {
	userID := ksuid.New().String()

	// Get the picture from the user info
	var picture *string
	var claims struct {
		Picture string `json:"picture"`
	}
	if err := userInfo.Claims(&claims); err == nil && claims.Picture != "" {
		picture = &claims.Picture
	}

	// create the user
	user := models.User{
		ID:           userID,
		PasswordHash: nil,
		Avatar:       picture,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// create the account
	account := models.Account{
		ID:             ksuid.New().String(),
		UserID:         userID,
		Provider:       provider,
		ProviderUserID: userInfo.Subject,
		Email:          userInfo.Email,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return user, account, nil
}

// GenerateUserAccountFromOAuthUserInfo generate the user and account from the oauth user info
func GenerateUserAccountFromOAuthUserInfo(userInfo *oidc.UserInfo, provider models.Provider, userID string) (models.Account, error) {
	// create the account
	account := models.Account{
		ID:             ksuid.New().String(),
		UserID:         userID,
		Provider:       provider,
		ProviderUserID: userInfo.Subject,
		Email:          userInfo.Email,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return account, nil
}

// GenerateSessionCookie generates a session cookie
func GenerateOAuthSessionCookie(session string) http.Cookie {
	oauthSessionCookie := http.Cookie{
		Name:     models.CookieNameOAuthSession,
		Value:    session,
		HttpOnly: true,
		Path:     "/api/auth/oauth",
		Secure:   config.Env().AppEnv == config.AppEnvProd,
		Expires:  time.Now().Add(time.Duration(config.Env().OAuthStateExpiresAt) * time.Second),
		SameSite: http.SameSiteLaxMode,
	}

	return oauthSessionCookie
}
