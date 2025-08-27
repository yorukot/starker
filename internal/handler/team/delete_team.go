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
// | Delete Team                                  |
// +----------------------------------------------+

// DeleteTeam godoc
// @Summary Delete a team
// @Description Deletes a team if the authenticated user is the owner
// @Tags team
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Success 200 {object} response.SuccessResponse "Team deleted successfully"
// @Failure 400 {object} response.ErrorResponse "Team ID is required"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 403 {object} response.ErrorResponse "Only team owner can delete the team"
// @Failure 404 {object} response.ErrorResponse "Team not found or user is not a member"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID} [delete]
// @Security BearerAuth
func (h *TeamHandler) DeleteTeam(w http.ResponseWriter, r *http.Request) {
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
		response.RespondWithError(w, http.StatusForbidden, "Only team owner can delete the team", "NOT_TEAM_OWNER")
		return
	}

	// Check if the team has any projects - prevent deletion if projects exist
	projectCount, err := repository.CountProjectsByTeamID(r.Context(), tx, teamID)
	if err != nil {
		zap.L().Error("Failed to count projects for team", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to check team projects", "FAILED_TO_CHECK_PROJECTS")
		return
	}

	if projectCount > 0 {
		response.RespondWithError(w, http.StatusBadRequest, "Cannot delete team with existing projects. Please delete all projects first.", "TEAM_HAS_PROJECTS")
		return
	}

	// Delete the team and all related team users
	if err = repository.DeleteTeamAndAllRelatedData(r.Context(), tx, teamID); err != nil {
		zap.L().Error("Failed to delete team", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to delete team", "FAILED_TO_DELETE_TEAM")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Response
	response.RespondWithJSON(w, http.StatusOK, nil)
}
