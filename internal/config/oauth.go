package config

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/yorukot/stargo/internal/models"
)

// OAuthConfig is the configuration for the OAuth providers
type OAuthConfig struct {
	Providers     map[models.Provider]*oauth2.Config
	OIDCProviders map[models.Provider]*oidc.Provider
}

// GetOAuthConfig returns the OAuth2 configuration for Google
func GetOAuthConfig() (*OAuthConfig, error) {
	googleOauthConfig := &oauth2.Config{
		RedirectURL:  Env().GoogleRedirectURL,
		ClientID:     Env().GoogleClientID,
		ClientSecret: Env().GoogleClientSecret,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}

	// Create OIDC provider for Google
	ctx := context.Background()
	googleOIDCProvider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		zap.L().Error("failed to create google oidc provider", zap.Error(err))
		return nil, err
	}

	return &OAuthConfig{
		Providers: map[models.Provider]*oauth2.Config{
			models.ProviderGoogle: googleOauthConfig,
		},
		OIDCProviders: map[models.Provider]*oidc.Provider{
			models.ProviderGoogle: googleOIDCProvider,
		},
	}, nil
}
