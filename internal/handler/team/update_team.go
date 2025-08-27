package team

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/response"
)

// +----------------------------------------------+
// | Update Team                                  |
// +----------------------------------------------+

type UpdateTeamRequest struct {
	Name string `json:"name" validate:"required,min=3,max=255"`
}

// UpdateTeam godoc
// @Summary Update a team
// @Description Updates a team's name if the authenticated user is the owner
// @Tags team
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param request body UpdateTeamRequest true "Update team request"
// @Success 200 {object} response.SuccessResponse{data=models.Team} "Team updated successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body or team ID is required"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 403 {object} response.ErrorResponse "Only team owner can update the team"
// @Failure 404 {object} response.ErrorResponse "Team not found or user is not a member"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID} [patch]
// @Security BearerAuth
func (h *TeamHandler) UpdateTeam(w http.ResponseWriter, r *http.Request) {
	// Get the team ID from the URL
	teamID := chi.URLParam(r, "teamID")
	if teamID == "" {
		response.RespondWithError(w, http.StatusBadRequest, "Team ID is required", "TEAM_ID_REQUIRED")
		return
	}

	// Decode the request body
	var updateTeamRequest UpdateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&updateTeamRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	// Validate the request body
	if err := validator.New().Struct(updateTeamRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	// Trim whitespace from name
	updateTeamRequest.Name = strings.TrimSpace(updateTeamRequest.Name)

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

	// Get the team to check if user is a member and get owner info
	team, err := repository.GetTeamByIDAndUserID(r.Context(), tx, teamID, userID)
	if err != nil {
		zap.L().Error("Failed to get team by ID and user ID", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get team", "FAILED_TO_GET_TEAM")
		return
	}

	// Check if team exists and user is a member
	if team == nil {
		response.RespondWithError(w, http.StatusNotFound, "Team not found or you are not a member", "TEAM_NOT_FOUND")
		return
	}

	// Check if the user is the owner of the team
	if team.OwnerID != userID {
		response.RespondWithError(w, http.StatusForbidden, "Only team owner can update the team", "NOT_TEAM_OWNER")
		return
	}

	// Update the team name
	if err = repository.UpdateTeam(r.Context(), tx, teamID, updateTeamRequest.Name); err != nil {
		zap.L().Error("Failed to update team", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to update team", "FAILED_TO_UPDATE_TEAM")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Update the team object with new name for response
	team.Name = updateTeamRequest.Name

	// Return the updated team object
	response.RespondWithJSON(w, http.StatusOK, team)
}
