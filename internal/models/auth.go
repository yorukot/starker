package models

import "time"

// CookieName constants
const (
	CookieNameOAuthSession = "oauth_session"
	CookieNameRefreshToken = "refresh_token"
)

// RefreshToken represents a refresh token for user authentication
type RefreshToken struct {
	ID        string     `json:"id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`                                                     // Unique identifier for the refresh token
	UserID    string     `json:"user_id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`                                                // User ID associated with this token
	Token     string     `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`                                     // The actual refresh token
	UserAgent *string    `json:"user_agent,omitempty" example:"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"` // User agent string from the client
	IP        *string    `json:"ip,omitempty" example:"192.168.1.100"`                                                        // IP address of the client
	UsedAt    *time.Time `json:"used_at,omitempty" example:"2023-01-01T12:00:00Z"`                                            // Timestamp when the token was last used
	CreatedAt time.Time  `json:"created_at" example:"2023-01-01T12:00:00Z"`                                                   // Timestamp when the token was created
}

// Provider represents the authentication provider type
type Provider string

// Provider constants
const (
	ProviderEmail  Provider = "email"  // Email/password authentication
	ProviderGoogle Provider = "google" // Google OAuth authentication
	// You can add more providers here
)
