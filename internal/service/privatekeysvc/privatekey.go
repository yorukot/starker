package privatekeysvc

import (
	"crypto/sha256"
	"encoding/base64"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/segmentio/ksuid"

	"github.com/yorukot/starker/internal/models"
)

type CreatePrivateKeyRequest struct {
	Name        string  `json:"name" validate:"required,min=3,max=255"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=500"`
	PrivateKey  string  `json:"private_key" validate:"required"`
}

// PrivateKeyValidate validates the create private key request
func PrivateKeyValidate(createPrivateKeyRequest CreatePrivateKeyRequest) error {
	return validator.New().Struct(createPrivateKeyRequest)
}

// GenerateFingerprint generates a SHA256 fingerprint for the private key
func GenerateFingerprint(privateKey string) string {
	hash := sha256.Sum256([]byte(privateKey))
	return "SHA256:" + base64.StdEncoding.EncodeToString(hash[:])
}

// GeneratePrivateKey generates a private key model for the create request
func GeneratePrivateKey(createPrivateKeyRequest CreatePrivateKeyRequest, teamID string) models.PrivateKey {
	now := time.Now()
	fingerprint := GenerateFingerprint(createPrivateKeyRequest.PrivateKey)

	return models.PrivateKey{
		ID:          ksuid.New().String(),
		TeamID:      teamID,
		Name:        createPrivateKeyRequest.Name,
		Description: createPrivateKeyRequest.Description,
		PrivateKey:  createPrivateKeyRequest.PrivateKey,
		Fingerprint: fingerprint,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
