package teamsvc

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/segmentio/ksuid"

	"github.com/yorukot/starker/internal/models"
)

type CreateTeamRequest struct {
	Name string `json:"name" validate:"required,min=3,max=255"`
}

// TeamValidate validate the register request
func TeamValidate(createTeamRequest CreateTeamRequest) error {
	return validator.New().Struct(createTeamRequest)
}

// GenerateTeam generate a team and team user for the create team request
func GenerateTeam(createTeamRequest CreateTeamRequest, ownerID string) (models.Team, models.TeamUser) {
	teamID := ksuid.New().String()
	now := time.Now()

	team := models.Team{
		ID:        teamID,
		OwnerID:   ownerID,
		Name:      createTeamRequest.Name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	teamUser := models.TeamUser{
		ID:        ksuid.New().String(),
		TeamID:    teamID,
		UserID:    ownerID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	return team, teamUser
}
