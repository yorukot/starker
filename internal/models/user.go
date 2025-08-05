package models

import "time"

// User represents a user in the system
type User struct {
	ID           string    `json:"id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`                   // Unique identifier for the user
	PasswordHash *string   `json:"password_hash,omitempty" example:"hashed_password"`         // Hashed password (omitted in responses)
	Avatar       *string   `json:"avatar,omitempty" example:"https://example.com/avatar.jpg"` // URL to user's avatar image
	CreatedAt    time.Time `json:"created_at" example:"2023-01-01T12:00:00Z"`                 // Timestamp when the user was created
	UpdatedAt    time.Time `json:"updated_at" example:"2023-01-01T12:00:00Z"`                 // Timestamp when the user was last updated
}

// Provider represents the authentication provider type
type Provider string

const (
	ProviderEmail  Provider = "email"  // Email/password authentication
	ProviderGoogle Provider = "google" // Google OAuth authentication
	// You can add more providers here
)

// Account represents how a user can login to the system
type Account struct {
	ID             string    `json:"id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`      // Unique identifier for the account
	Provider       Provider  `json:"provider" example:"email"`                     // Authentication provider type
	ProviderUserID string    `json:"provider_user_id" example:"user123"`           // User ID from the provider
	UserID         string    `json:"user_id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"` // Associated user ID
	Email          string    `json:"email" example:"user@example.com"`             // User's email address
	CreatedAt      time.Time `json:"created_at" example:"2023-01-01T12:00:00Z"`    // Timestamp when the account was created
	UpdatedAt      time.Time `json:"updated_at" example:"2023-01-01T12:00:00Z"`    // Timestamp when the account was last updated
}

// OAuthToken represents OAuth tokens for external providers
type OAuthToken struct {
	AccountID    string    `json:"account_id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`     // Associated account ID
	AccessToken  string    `json:"access_token" example:"ya29.a0AfH6SMC..."`            // OAuth access token
	RefreshToken string    `json:"refresh_token" example:"1//0GWthXqhYjIsKCgYIARAA..."` // OAuth refresh token
	Expiry       time.Time `json:"expiry" example:"2023-01-01T13:00:00Z"`               // Token expiration time
	TokenType    string    `json:"token_type" example:"Bearer"`                         // Token type (usually Bearer)
	Provider     Provider  `json:"provider" example:"google"`                           // OAuth provider
	CreatedAt    time.Time `json:"created_at" example:"2023-01-01T12:00:00Z"`           // Timestamp when the token was created
	UpdatedAt    time.Time `json:"updated_at" example:"2023-01-01T12:00:00Z"`           // Timestamp when the token was last updated
}
