package encrypt

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/segmentio/ksuid"
)

// GenerateSecureRefreshToken generate a secure refresh token
func GenerateSecureRefreshToken() (string, error) {
	bytes := make([]byte, 256) // 256-bit
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return ksuid.New().String() + "_" + base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes), nil
}
