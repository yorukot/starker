package project

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/internal/service/projectsvc"
	"github.com/yorukot/starker/pkg/response"
)

// +----------------------------------------------+
// | Update Project                              |
// +----------------------------------------------+

// UpdateProject godoc
// @Summary Update a project
// @Description Updates an existing project within a team
// @Tags project
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param projectID path string true "Project ID"
// @Param request body projectsvc.UpdateProjectRequest true "Project update request"
// @Success 200 {object} response.SuccessResponse{data=models.Project} "Project updated successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body or team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 404 {object} response.ErrorResponse "Project not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/projects/{projectID} [put]
// @Security BearerAuth
func (h *ProjectHandler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	// Get the team ID and project ID from the URL parameters
	teamID := chi.URLParam(r, "teamID")
	projectID := chi.URLParam(r, "projectID")

	// Parse the request body into the update project request struct
	var updateProjectRequest projectsvc.UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&updateProjectRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	// Validate the project update request
	if err := projectsvc.ProjectUpdateValidate(updateProjectRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

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

	// Update the project in the database
	project, err := repository.UpdateProject(r.Context(), tx, teamID, projectID, updateProjectRequest)
	if err != nil {
		zap.L().Error("Failed to update project", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to update project", "PROJECT_UPDATE_FAILED")
		return
	}

	// Check if project was found and updated
	if project == nil {
		response.RespondWithError(w, http.StatusNotFound, "Project not found", "PROJECT_NOT_FOUND")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Return success response with the updated project
	response.RespondWithJSON(w, http.StatusOK, project)
}
