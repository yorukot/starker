package authsvc

import (
	"fmt"

	"github.com/yorukot/stargo/internal/models"
)

// ParseProvider parse the provider from the request
func ParseProvider(provider string) (models.Provider, error) {
	switch provider {
	case string(models.ProviderGoogle):
		return models.ProviderGoogle, nil
	default:
		return "", fmt.Errorf("invalid provider: %s", provider)
	}
}
