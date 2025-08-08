package encrypt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTSecret is the secret for the JWT
// We doing this because this make the function more testable
type JWTSecret struct {
	Secret string
}

// AccessTokenClaims is the claims for the access token
type AccessTokenClaims struct {
	Issuer    string `json:"iss"`
	Subject   string `json:"sub"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
}

// GenerateAccessToken generate an access token
func (j *JWTSecret) GenerateAccessToken(issuer string, subject string, expiresAt time.Time) (string, error) {
	claims := AccessTokenClaims{
		Issuer:    issuer,
		Subject:   subject,
		ExpiresAt: expiresAt.Unix(),
		IssuedAt:  time.Now().Unix(),
	}

	// TODO: Maybe need a way to covert the struct to map
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": claims.Issuer,
		"sub": claims.Subject,
		"exp": claims.ExpiresAt,
		"iat": claims.IssuedAt,
	})

	return token.SignedString([]byte(j.Secret))
}

// ValidateAccessTokenAndGetClaims validate the access token and get the claims
func (j *JWTSecret) ValidateAccessTokenAndGetClaims(token string) (bool, AccessTokenClaims, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.Secret), nil
	})

	if err != nil {
		if err == jwt.ErrTokenInvalidClaims {
			return false, AccessTokenClaims{}, nil
		}

		return false, AccessTokenClaims{}, err
	}

	// TODO: Maybe need a more clean way to covert the map to struct
	issuer, ok := claims["iss"].(string)
	if !ok {
		return false, AccessTokenClaims{}, nil
	}

	subject, ok := claims["sub"].(string)
	if !ok {
		return false, AccessTokenClaims{}, nil
	}

	expiresAt, ok := claims["exp"].(float64)
	if !ok {
		return false, AccessTokenClaims{}, nil
	}

	issuedAt, ok := claims["iat"].(float64)
	if !ok {
		return false, AccessTokenClaims{}, nil
	}

	accessTokenClaims := AccessTokenClaims{
		Issuer:    issuer,
		Subject:   subject,
		ExpiresAt: int64(expiresAt),
		IssuedAt:  int64(issuedAt),
	}

	return true, accessTokenClaims, nil
}

// OAuthStateClaims is the claims for the oauth state
type OAuthStateClaims struct {
	State       string `json:"state"`
	RedirectURI string `json:"redirect_uri"`
	ExpiresAt   int64  `json:"exp"`
	Subject     string `json:"sub"`
}

// GenerateOAuthState generate an oauth state
func (j *JWTSecret) GenerateOAuthState(state string, redirectURI string, expiresAt time.Time, userID string) (string, error) {
	claims := OAuthStateClaims{
		State:       state,
		RedirectURI: redirectURI,
		ExpiresAt:   expiresAt.Unix(),
		Subject:     userID,
	}

	// TODO: Maybe need a way to covert the struct to map
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":          claims.Subject,
		"state":        claims.State,
		"redirect_uri": claims.RedirectURI,
		"exp":          claims.ExpiresAt,
	})

	return token.SignedString([]byte(j.Secret))
}

// ValidateOAuthStateAndGetClaims validate the oauth state and get the claims
func (j *JWTSecret) ValidateOAuthStateAndGetClaims(token string) (bool, OAuthStateClaims, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.Secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenInvalidClaims) {
			return false, OAuthStateClaims{}, nil
		}

		return false, OAuthStateClaims{}, err
	}

	// TODO: Maybe need a more clean way to covert the map to struct
	state, ok := claims["state"].(string)
	if !ok {
		return false, OAuthStateClaims{}, nil
	}

	redirectURI, ok := claims["redirect_uri"].(string)
	if !ok {
		return false, OAuthStateClaims{}, nil
	}

	expiresAt, ok := claims["exp"].(float64)
	if !ok {
		return false, OAuthStateClaims{}, nil
	}

	oauthStateClaims := OAuthStateClaims{
		State:       state,
		RedirectURI: redirectURI,
		ExpiresAt:   int64(expiresAt),
	}

	return true, oauthStateClaims, nil
}
