package repository

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/yorukot/starker/internal/models"
)

// GetTeamsByUserID gets all teams that a user is a member of
func GetTeamsByUserID(ctx context.Context, db pgx.Tx, userID string) ([]models.Team, error) {
	query := `
		SELECT t.id, t.owner_id, t.name, t.created_at, t.updated_at
		FROM teams t
		INNER JOIN team_users tu ON t.id = tu.team_id
		WHERE tu.user_id = $1
		ORDER BY t.created_at DESC
	`
	rows, err := db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []models.Team
	for rows.Next() {
		var team models.Team
		err := rows.Scan(
			&team.ID,
			&team.OwnerID,
			&team.Name,
			&team.CreatedAt,
			&team.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return teams, nil
}

// GetTeamByIDAndUserID gets a specific team if the user is a member of it
func GetTeamByIDAndUserID(ctx context.Context, db pgx.Tx, teamID, userID string) (*models.Team, error) {
	query := `
		SELECT t.id, t.owner_id, t.name, t.created_at, t.updated_at
		FROM teams t
		INNER JOIN team_users tu ON t.id = tu.team_id
		WHERE t.id = $1 AND tu.user_id = $2
	`
	var team models.Team
	err := db.QueryRow(ctx, query, teamID, userID).Scan(
		&team.ID,
		&team.OwnerID,
		&team.Name,
		&team.CreatedAt,
		&team.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &team, nil
}

// CheckTeamAccess verifies if a user has access to a team
func CheckTeamAccess(ctx context.Context, db pgx.Tx, teamID, userID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM team_users
			WHERE team_id = $1 AND user_id = $2
		)
	`
	var exists bool
	err := db.QueryRow(ctx, query, teamID, userID).Scan(&exists)
	return exists, err
}

// CreateTeam creates a new team
func CreateTeam(ctx context.Context, db pgx.Tx, team models.Team) error {
	query := `
		INSERT INTO teams (id, owner_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := db.Exec(ctx, query,
		team.ID,
		team.OwnerID,
		team.Name,
		team.CreatedAt,
		team.UpdatedAt,
	)
	return err
}

// CreateTeamUser creates a new team user relationship
func CreateTeamUser(ctx context.Context, db pgx.Tx, teamUser models.TeamUser) error {
	query := `
		INSERT INTO team_users (id, team_id, user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := db.Exec(ctx, query,
		teamUser.ID,
		teamUser.TeamID,
		teamUser.UserID,
		teamUser.CreatedAt,
		teamUser.UpdatedAt,
	)
	return err
}

// CreateTeamAndTeamUser creates a team and adds the owner as a team user
func CreateTeamAndTeamUser(ctx context.Context, db pgx.Tx, team models.Team, teamUser models.TeamUser) error {
	if err := CreateTeam(ctx, db, team); err != nil {
		return err
	}
	if err := CreateTeamUser(ctx, db, teamUser); err != nil {
		return err
	}
	return nil
}

// DeleteTeam deletes a team
func DeleteTeam(ctx context.Context, db pgx.Tx, teamID string) error {
	query := `DELETE FROM teams WHERE id = $1`
	_, err := db.Exec(ctx, query, teamID)
	return err
}

// DeleteTeamAndAllRelatedData deletes a team and all related data (invites, users, etc.)
func DeleteTeamAndAllRelatedData(ctx context.Context, db pgx.Tx, teamID string) error {
	// Delete team invites first
	if err := deleteTeamInvites(ctx, db, teamID); err != nil {
		return err
	}
	
	// Delete team users
	if err := deleteTeamUsers(ctx, db, teamID); err != nil {
		return err
	}
	
	// Delete private keys
	if err := deleteTeamPrivateKeys(ctx, db, teamID); err != nil {
		return err
	}
	
	// Delete servers
	if err := deleteTeamServers(ctx, db, teamID); err != nil {
		return err
	}
	
	// Finally delete the team
	if err := DeleteTeam(ctx, db, teamID); err != nil {
		return err
	}
	return nil
}

// deleteTeamInvites deletes all invites for a team
func deleteTeamInvites(ctx context.Context, db pgx.Tx, teamID string) error {
	query := `DELETE FROM team_invites WHERE team_id = $1`
	_, err := db.Exec(ctx, query, teamID)
	return err
}

// deleteTeamUsers deletes all team user relationships for a team
func deleteTeamUsers(ctx context.Context, db pgx.Tx, teamID string) error {
	query := `DELETE FROM team_users WHERE team_id = $1`
	_, err := db.Exec(ctx, query, teamID)
	return err
}

// deleteTeamPrivateKeys deletes all private keys for a team
func deleteTeamPrivateKeys(ctx context.Context, db pgx.Tx, teamID string) error {
	query := `DELETE FROM private_keys WHERE team_id = $1`
	_, err := db.Exec(ctx, query, teamID)
	return err
}

// deleteTeamServers deletes all servers for a team
func deleteTeamServers(ctx context.Context, db pgx.Tx, teamID string) error {
	query := `DELETE FROM servers WHERE team_id = $1`
	_, err := db.Exec(ctx, query, teamID)
	return err
}
