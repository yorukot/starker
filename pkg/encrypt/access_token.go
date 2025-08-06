package encrypt

import (
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
		return false, AccessTokenClaims{}, err
	}

	accessTokenClaims := AccessTokenClaims{
		Issuer:    claims["iss"].(string),
		Subject:   claims["sub"].(string),
		ExpiresAt: int64(claims["exp"].(float64)),
		IssuedAt:  int64(claims["iat"].(float64)),
	}

	return true, accessTokenClaims, nil
}
