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
// | Create Project                               |
// +----------------------------------------------+

// CreateProject godoc
// @Summary Create a new project
// @Description Creates a new project within a team for managing deployments and configurations
// @Tags project
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param request body projectsvc.CreateProjectRequest true "Project creation request"
// @Success 201 {object} response.SuccessResponse{data=models.Project} "Project created successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body or team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/projects [post]
// @Security BearerAuth
func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	// Get the team ID from the URL parameter
	teamID := chi.URLParam(r, "teamID")

	// Parse the request body into the create project request struct
	var createProjectRequest projectsvc.CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&createProjectRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	// Validate the project creation request
	if err := projectsvc.ProjectValidate(createProjectRequest); err != nil {
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

	// Create the project in the database
	project, err := repository.CreateProject(r.Context(), tx, teamID, createProjectRequest)
	if err != nil {
		zap.L().Error("Failed to create project", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to create project", "PROJECT_CREATION_FAILED")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Return success response with the created project
	response.RespondWithJSON(w, http.StatusCreated, "Project created successfully", project)
}
