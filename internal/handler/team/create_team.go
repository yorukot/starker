package team

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/response"
)

// +----------------------------------------------+
// | Create Team                                  |
// +----------------------------------------------+

type createTeamRequest struct {
	Name string `json:"name" validate:"required,min=3,max=255"`
}

// CreateTeam godoc
// @Summary Create a new team
// @Description Creates a new team with the authenticated user as the owner
// @Tags team
// @Accept json
// @Produce json
// @Param request body teamsvc.CreateTeamRequest true "Create team request"
// @Success 200 {object} response.SuccessResponse "Team created successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams [post]
// @Security BearerAuth
func (h *TeamHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	// Decode the request body
	var createTeamRequest createTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&createTeamRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	// Validate the request body
	if err := validator.New().Struct(createTeamRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	// Get user ID from context
	userID := r.Context().Value(middleware.UserIDKey).(string)

	// Begin the transaction
	tx, err := repository.StartTransaction(h.DB, r.Context())
	if err != nil {
		zap.L().Error("Failed to begin transaction", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
		return
	}

	defer repository.DeferRollback(tx, r.Context())

	// Generate the team and team user
	team, teamUser := generateTeam(createTeamRequest, userID)

	// Create the team and team user in the database
	if err = repository.CreateTeamAndTeamUser(r.Context(), tx, team, teamUser); err != nil {
		zap.L().Error("Failed to create team", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to create team", "FAILED_TO_CREATE_TEAM")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Respond with the team object
	response.RespondWithJSON(w, http.StatusOK, team)
}

// generateTeam generate a team and team user for the create team request
func generateTeam(createTeamRequest createTeamRequest, ownerID string) (models.Team, models.TeamUser) {
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
