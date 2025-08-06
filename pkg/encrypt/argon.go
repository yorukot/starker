package encrypt

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// params is the parameters for the Argon2id hash
type params struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

// CreateArgon2idHash generate a Argon2id hash for the password
func CreateArgon2idHash(password string) (string, error) {
	p := &params{
		memory:      128 * 1024,
		iterations:  15,
		parallelism: 4,
		saltLength:  16,
		keyLength:   32,
	}

	salt := make([]byte, p.saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, p.iterations, p.memory, p.parallelism, p.keyLength)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encodedHash := fmt.Sprintf("$argon2id$v=19$t=%d$m=%d$p=%d$%s$%s", p.iterations, p.memory, p.parallelism, b64Salt, b64Hash)

	return encodedHash, nil
}

// ComparePasswordAndHash compare the password and the hash
func ComparePasswordAndHash(password, encodedHash string) (match bool, err error) {
	p, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	otherHash := argon2.IDKey([]byte(password), salt, p.iterations, p.memory, p.parallelism, p.keyLength)

	// Use subtle.ConstantTimeCompare to avoid timing attacks
	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return true, nil
	}
	return false, nil
}

// decodeHash decode the hash
func decodeHash(encodedHash string) (p *params, salt, hash []byte, err error) {
	vals := strings.Split(encodedHash, "$")
	if len(vals) != 8 {
		return nil, nil, nil, fmt.Errorf("invalid hash")
	}

	var version int
	_, err = fmt.Sscanf(vals[2], "v=%d", &version)
	if err != nil {
		return nil, nil, nil, err
	}
	if version != argon2.Version {
		return nil, nil, nil, fmt.Errorf("incompatible version")
	}

	p = &params{}
	_, err = fmt.Sscanf(vals[3], "t=%d", &p.iterations)
	if err != nil {
		return nil, nil, nil, err
	}
	_, err = fmt.Sscanf(vals[4], "m=%d", &p.memory)
	if err != nil {
		return nil, nil, nil, err
	}
	_, err = fmt.Sscanf(vals[5], "p=%d", &p.parallelism)
	if err != nil {
		return nil, nil, nil, err
	}

	salt, err = base64.RawStdEncoding.Strict().DecodeString(vals[6])
	if err != nil {
		return nil, nil, nil, err
	}
	p.saltLength = uint32(len(salt))

	hash, err = base64.RawStdEncoding.Strict().DecodeString(vals[7])
	if err != nil {
		return nil, nil, nil, err
	}
	p.keyLength = uint32(len(hash))

	return p, salt, hash, nil
}
