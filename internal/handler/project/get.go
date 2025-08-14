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
// | Get Projects                                |
// +----------------------------------------------+

// GetProjects godoc
// @Summary Get all projects in a team
// @Description Retrieves all projects that belong to a specific team
// @Tags project
// @Produce json
// @Param teamID path string true "Team ID"
// @Success 200 {object} response.SuccessResponse{data=[]models.Project} "Projects retrieved successfully"
// @Failure 400 {object} response.ErrorResponse "Team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/projects [get]
// @Security BearerAuth
func (h *ProjectHandler) GetProjects(w http.ResponseWriter, r *http.Request) {
	// Get the team ID from the URL parameter
	teamID := chi.URLParam(r, "teamID")

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

	// Get all projects for the team
	projects, err := repository.GetProjects(r.Context(), tx, teamID)
	if err != nil {
		zap.L().Error("Failed to get projects", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get projects", "FAILED_TO_GET_PROJECTS")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Return success response with the projects
	response.RespondWithJSON(w, http.StatusOK, "Projects retrieved successfully", projects)
}

// +----------------------------------------------+
// | Get Single Project                          |
// +----------------------------------------------+

// GetProject godoc
// @Summary Get a specific project
// @Description Retrieves a specific project by its ID within a team
// @Tags project
// @Produce json
// @Param teamID path string true "Team ID"
// @Param projectID path string true "Project ID"
// @Success 200 {object} response.SuccessResponse{data=models.Project} "Project retrieved successfully"
// @Failure 400 {object} response.ErrorResponse "Team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 404 {object} response.ErrorResponse "Project not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/projects/{projectID} [get]
// @Security BearerAuth
func (h *ProjectHandler) GetProject(w http.ResponseWriter, r *http.Request) {
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

	// Get the specific project
	project, err := repository.GetProject(r.Context(), tx, teamID, projectID)
	if err != nil {
		zap.L().Error("Failed to get project", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get project", "FAILED_TO_GET_PROJECT")
		return
	}

	// Check if project was found
	if project == nil {
		response.RespondWithError(w, http.StatusNotFound, "Project not found", "PROJECT_NOT_FOUND")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Return success response with the project
	response.RespondWithJSON(w, http.StatusOK, "Project retrieved successfully", project)
}
