package encrypt

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	mathRand "math/rand"

	"github.com/segmentio/ksuid"
)

// GenerateRandomString generate a random string
func GenerateRandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[num.Int64()]
	}

	return string(result), nil
}

// GenerateSecureRefreshToken generate a secure refresh token
func GenerateSecureRefreshToken() (string, error) {
	bytes := make([]byte, 256) // 256-bit
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return ksuid.New().String() + "_" + base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes), nil
}

// GenerateRandomUserDisplayName generate a random user display name
func GenerateRandomUserDisplayName() string {
	names := []string{
		"Rodney",
		"Osodo",
		"David",
		"RodneyOsodo",
		"DavidOsodo",
		"OsodoRodney",
		"Rodnavi",
		"Osodney",
		"DaveOsodo",
		"Rododo",
		"Osavid",
		"R.O.D.",
		"OsodoX",
		"D-Rod",
		"Rodnix",
		"Osodroid",
		"Davrod",
		"Rodnardo",
		"OsoDave",
		"D-Rodney",
		"Rodnova",
		"Osodash",
		"DaveyOso",
		"Rodavido",
		"OsoDyn",
		"Rodnator",
		"Osodino",
		"Davney",
		"RodoDave",
		"OsoRod",
		"RodVids",
		"Osodark",
		"DaveRod",
		"RodZone",
		"OsoNova",
		"RodVido",
		"Osodex",
		"RodVader",
		"OsoDrift",
		"Davodo",
		"RodOso",
		"OsoVid",
		"RodStar",
		"Osodream",
		"DaveNova",
		"RodBlaze",
		"OsoKnight",
	}

	randomIndex := mathRand.Intn(len(names))
	randomName := names[randomIndex]

	return randomName
}
