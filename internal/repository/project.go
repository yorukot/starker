package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/service/projectsvc"
)

// GetProjects gets all projects for a team
func GetProjects(ctx context.Context, db pgx.Tx, teamID string) ([]models.Project, error) {
	query := `
		SELECT id, team_id, name, description, created_at, updated_at
		FROM projects
		WHERE team_id = $1
		ORDER BY created_at DESC
	`
	rows, err := db.Query(ctx, query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var project models.Project
		err := rows.Scan(
			&project.ID,
			&project.TeamID,
			&project.Name,
			&project.Description,
			&project.CreatedAt,
			&project.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}

	return projects, rows.Err()
}

// GetProject gets a project by ID and team ID
func GetProject(ctx context.Context, db pgx.Tx, teamID, projectID string) (*models.Project, error) {
	query := `
		SELECT id, team_id, name, description, created_at, updated_at
		FROM projects
		WHERE id = $1 AND team_id = $2
	`
	var project models.Project
	err := db.QueryRow(ctx, query, projectID, teamID).Scan(
		&project.ID,
		&project.TeamID,
		&project.Name,
		&project.Description,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &project, nil
}

// CreateProject creates a new project
func CreateProject(ctx context.Context, db pgx.Tx, teamID string, createProjectRequest projectsvc.CreateProjectRequest) (*models.Project, error) {
	project := projectsvc.GenerateProject(createProjectRequest, teamID)

	query := `
		INSERT INTO projects (id, team_id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := db.Exec(ctx, query,
		project.ID,
		project.TeamID,
		project.Name,
		project.Description,
		project.CreatedAt,
		project.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &project, nil
}

// UpdateProject updates an existing project
func UpdateProject(ctx context.Context, db pgx.Tx, teamID, projectID string, updateProjectRequest projectsvc.UpdateProjectRequest) (*models.Project, error) {
	// First check if project exists
	existingProject, err := GetProject(ctx, db, teamID, projectID)
	if err != nil {
		return nil, err
	}
	if existingProject == nil {
		return nil, nil
	}

	// Update the project with new values
	updatedProject := projectsvc.UpdateProjectFromRequest(*existingProject, updateProjectRequest)

	query := `
		UPDATE projects
		SET name = $1, description = $2, updated_at = $3
		WHERE id = $4 AND team_id = $5
	`
	_, err = db.Exec(ctx, query,
		updatedProject.Name,
		updatedProject.Description,
		updatedProject.UpdatedAt,
		projectID,
		teamID,
	)
	if err != nil {
		return nil, err
	}

	return &updatedProject, nil
}

// DeleteProject deletes a project by ID and team ID
func DeleteProject(ctx context.Context, db pgx.Tx, teamID, projectID string) (bool, error) {
	query := `
		DELETE FROM projects
		WHERE id = $1 AND team_id = $2
	`
	result, err := db.Exec(ctx, query, projectID, teamID)
	if err != nil {
		return false, err
	}

	rowsAffected := result.RowsAffected()
	return rowsAffected > 0, nil
}
