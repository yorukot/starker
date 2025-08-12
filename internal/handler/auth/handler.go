package auth

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yorukot/starker/internal/config"
)

// AuthHandler is the handler for the auth routes
type AuthHandler struct {
	DB *pgxpool.Pool
}

// OAuthHandler is the handler for the oauth routes
type OAuthHandler struct {
	DB          *pgxpool.Pool
	OAuthConfig *config.OAuthConfig
}
