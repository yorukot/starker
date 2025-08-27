package team

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/response"
)

// +----------------------------------------------+
// | Get Team                                     |
// +----------------------------------------------+

// GetTeam godoc
// @Summary Get a specific team
// @Description Gets a specific team by ID if the authenticated user is a member of it
// @Tags team
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Success 200 {object} response.SuccessResponse{data=models.Team} "Team retrieved successfully"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 404 {object} response.ErrorResponse "Team not found or user is not a member"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID} [get]
// @Security BearerAuth
func (h *TeamHandler) GetTeam(w http.ResponseWriter, r *http.Request) {
	// Get the team ID from the URL
	teamID := chi.URLParam(r, "teamID")
	if teamID == "" {
		response.RespondWithError(w, http.StatusBadRequest, "Team ID is required", "TEAM_ID_REQUIRED")
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

	// Get the team from the database (Will also check if the user is a member of the team in sql)
	// We doing this by use join so first we get the data from the team_users and then join the team data.
	team, err := repository.GetTeamByIDAndUserID(r.Context(), tx, teamID, userID)
	if err != nil {
		zap.L().Error("Failed to get team by ID and user ID", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get team", "FAILED_TO_GET_TEAM")
		return
	}

	// If we find the team, we return the team object, otherwise we return a 404
	if team == nil {
		response.RespondWithError(w, http.StatusNotFound, "Team not found or you are not a member", "TEAM_NOT_FOUND")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Return the team object
	response.RespondWithJSON(w, http.StatusOK, team)
}
