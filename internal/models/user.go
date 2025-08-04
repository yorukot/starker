package models

import "time"

// User is the user model
type User struct {
	ID        string    `json:"id"`
	Password  *string   `json:"password"`
	Avatar    *string   `json:"avatar"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Provider is the provider type this can be email or google or apple or whatever
type Provider string

const (
	ProviderEmail  Provider = "email"
	ProviderGoogle Provider = "google"
	// You can add more providers here
)

// Account mean how does the user login to the system
type Account struct {
	ID             string    `json:"id"`
	Provider       Provider  `json:"provider"`
	ProviderUserID string    `json:"provider_user_id"`
	UserID         string    `json:"user_id"`
	Email          string    `json:"email"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// OAuthToken is the token for the OAuth provider
type OAuthToken struct {
	AccountID    string    `json:"account_id"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	Expiry       time.Time `json:"expiry"`
	TokenType    string    `json:"token_type"`
	Provider     Provider  `json:"provider"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
