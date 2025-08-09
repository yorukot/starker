package auth

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/internal/service/authsvc"
)

// GenerateTokenAndSaveRefreshToken generate a refresh token and save it to the database
func GenerateTokenAndSaveRefreshToken(ctx context.Context, db pgx.Tx, userID string, userAgent string, ip string) (models.RefreshToken, error) {
	refreshToken, err := authsvc.GenerateRefreshToken(userID, userAgent, ip)
	if err != nil {
		return models.RefreshToken{}, err
	}

	err = repository.CreateRefreshToken(ctx, db, refreshToken)
	if err != nil {
		return models.RefreshToken{}, err
	}

	return refreshToken, nil
}
