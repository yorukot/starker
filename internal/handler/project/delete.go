package project

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/response"
)

// +----------------------------------------------+
// | Delete Project                              |
// +----------------------------------------------+

// DeleteProject godoc
// @Summary Delete a project
// @Description Deletes an existing project from a team
// @Tags project
// @Produce json
// @Param teamID path string true "Team ID"
// @Param projectID path string true "Project ID"
// @Success 200 {object} response.SuccessResponse "Project deleted successfully"
// @Failure 400 {object} response.ErrorResponse "Team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 404 {object} response.ErrorResponse "Project not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/projects/{projectID} [delete]
// @Security BearerAuth
func (h *ProjectHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	// Get the team ID and project ID from the URL parameters
	teamID := chi.URLParam(r, "teamID")
	projectID := chi.URLParam(r, "projectID")

	// Get user ID from context
	userID := r.Context().Value(middleware.UserIDKey).(string)

	// Start the transaction
	tx, err := repository.StartTransaction(h.DB, r.Context())
	if err != nil {
		zap.L().Error("Failed to begin transaction", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
		return
	}
	defer repository.DeferRollback(tx, r.Context())

	// Check if user has access to the team
	hasAccess, err := repository.CheckTeamAccess(r.Context(), tx, teamID, userID)
	if err != nil {
		zap.L().Error("Failed to check team access", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to check team access", "FAILED_TO_CHECK_TEAM_ACCESS")
		return
	}
	if !hasAccess {
		response.RespondWithError(w, http.StatusBadRequest, "Team access denied", "TEAM_ACCESS_DENIED")
		return
	}

	// Delete the project from the database
	deleted, err := repository.DeleteProject(r.Context(), tx, teamID, projectID)
	if err != nil {
		zap.L().Error("Failed to delete project", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to delete project", "PROJECT_DELETE_FAILED")
		return
	}

	// Check if project was found and deleted
	if !deleted {
		response.RespondWithError(w, http.StatusNotFound, "Project not found", "PROJECT_NOT_FOUND")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Return success response
	response.RespondWithJSON(w, http.StatusOK, "Project deleted successfully", nil)
}
