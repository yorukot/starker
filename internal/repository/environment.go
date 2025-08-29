package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/yorukot/starker/internal/models"
)

// +----------------------------------------------+
// | Service Environment Functions                |
// +----------------------------------------------+

// GetServiceEnvironments gets all environment variables for a service
func GetServiceEnvironments(ctx context.Context, db pgx.Tx, serviceID string) ([]models.ServiceEnvironment, error) {
	query := `
		SELECT id, service_id, key, value, created_at, updated_at
		FROM service_environments
		WHERE service_id = $1
		ORDER BY key ASC
	`
	rows, err := db.Query(ctx, query, serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var environments []models.ServiceEnvironment
	for rows.Next() {
		var env models.ServiceEnvironment
		err := rows.Scan(
			&env.ID,
			&env.ServiceID,
			&env.Key,
			&env.Value,
			&env.CreatedAt,
			&env.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		environments = append(environments, env)
	}

	return environments, rows.Err()
}

// GetServiceEnvironment gets a single environment variable by ID and service ID
func GetServiceEnvironment(ctx context.Context, db pgx.Tx, id int64, serviceID string) (*models.ServiceEnvironment, error) {
	query := `
		SELECT id, service_id, key, value, created_at, updated_at
		FROM service_environments
		WHERE id = $1 AND service_id = $2
	`
	var env models.ServiceEnvironment
	err := db.QueryRow(ctx, query, id, serviceID).Scan(
		&env.ID,
		&env.ServiceID,
		&env.Key,
		&env.Value,
		&env.CreatedAt,
		&env.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("environment variable not found")
		}
		return nil, err
	}

	return &env, nil
}

// CreateServiceEnvironment creates a new environment variable
func CreateServiceEnvironment(ctx context.Context, db pgx.Tx, env models.ServiceEnvironment) error {
	query := `
		INSERT INTO service_environments (service_id, key, value, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	err := db.QueryRow(ctx, query,
		env.ServiceID,
		env.Key,
		env.Value,
		env.CreatedAt,
		env.UpdatedAt,
	).Scan(&env.ID)
	return err
}

// CreateServiceEnvironments creates multiple environment variables in batch
func CreateServiceEnvironments(ctx context.Context, db pgx.Tx, environments []models.ServiceEnvironment) error {
	if len(environments) == 0 {
		return nil
	}

	query := `
		INSERT INTO service_environments (service_id, key, value, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	for _, env := range environments {
		_, err := db.Exec(ctx, query,
			env.ServiceID,
			env.Key,
			env.Value,
			env.CreatedAt,
			env.UpdatedAt,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateServiceEnvironment updates an existing environment variable
func UpdateServiceEnvironment(ctx context.Context, db pgx.Tx, env models.ServiceEnvironment) error {
	query := `
		UPDATE service_environments
		SET key = $3, value = $4, updated_at = $5
		WHERE id = $1 AND service_id = $2
	`
	result, err := db.Exec(ctx, query,
		env.ID,
		env.ServiceID,
		env.Key,
		env.Value,
		env.UpdatedAt,
	)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("environment variable not found")
	}

	return nil
}

// UpdateServiceEnvironments updates multiple environment variables in batch
func UpdateServiceEnvironments(ctx context.Context, db pgx.Tx, environments []models.ServiceEnvironment) error {
	if len(environments) == 0 {
		return nil
	}

	query := `
		UPDATE service_environments
		SET key = $3, value = $4, updated_at = $5
		WHERE id = $1 AND service_id = $2
	`

	for _, env := range environments {
		result, err := db.Exec(ctx, query,
			env.ID,
			env.ServiceID,
			env.Key,
			env.Value,
			env.UpdatedAt,
		)
		if err != nil {
			return err
		}

		rowsAffected := result.RowsAffected()
		if rowsAffected == 0 {
			return errors.New("environment variable not found")
		}
	}
	return nil
}

// DeleteServiceEnvironment deletes a single environment variable
func DeleteServiceEnvironment(ctx context.Context, db pgx.Tx, id int64, serviceID string) error {
	query := `DELETE FROM service_environments WHERE id = $1 AND service_id = $2`
	result, err := db.Exec(ctx, query, id, serviceID)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("environment variable not found")
	}

	return nil
}

// DeleteServiceEnvironments deletes all environment variables for a service
func DeleteServiceEnvironments(ctx context.Context, db pgx.Tx, serviceID string) error {
	query := `DELETE FROM service_environments WHERE service_id = $1`
	_, err := db.Exec(ctx, query, serviceID)
	return err
}
