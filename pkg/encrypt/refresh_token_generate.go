package encrypt

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func GenerateSecureRefreshToken() (string, error) {
	bytes := make([]byte, 32) // 256-bit
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes), nil
}
