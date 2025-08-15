package service

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/internal/service/servicesvc"
	"github.com/yorukot/starker/pkg/response"
)

// +----------------------------------------------+
// | Create Service                               |
// +----------------------------------------------+

// CreateService godoc
// @Summary Create a new service
// @Description Creates a new service with Docker compose configuration within a team and project
// @Tags service
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param projectID path string true "Project ID"
// @Param request body servicesvc.CreateServiceRequest true "Service creation request"
// @Success 201 {object} response.SuccessResponse{data=models.Service} "Service created successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body, team access denied, or project not found"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/projects/{projectID}/services [post]
// @Security BearerAuth
func (h *ServiceHandler) CreateService(w http.ResponseWriter, r *http.Request) {
	// Get teamID and projectID from the request
	teamID := chi.URLParam(r, "teamID")
	projectID := chi.URLParam(r, "projectID")

	// Get the service request from the request body
	var createServiceRequest servicesvc.CreateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&createServiceRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	// Validate the request body
	if err := servicesvc.ServiceValidate(createServiceRequest); err != nil {
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

	// Check if the user has access to the team
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

	// Check if the project exists
	project, err := repository.GetProject(r.Context(), tx, teamID, projectID)
	if err != nil {
		zap.L().Error("Failed to get project", zap.Error(err))
		response.RespondWithError(w, http.StatusBadRequest, "Project not found", "PROJECT_NOT_FOUND")
		return
	}

	// Generate service model
	service := servicesvc.GenerateService(createServiceRequest, teamID, createServiceRequest.ServerID, project.ID)

	// Generate compose config model
	composeConfig := servicesvc.GenerateServiceComposeConfig(service.ID, createServiceRequest.ComposeFile)

	// Create the service
	if err = repository.CreateService(r.Context(), tx, service); err != nil {
		zap.L().Error("Failed to create service", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to create service", "FAILED_TO_CREATE_SERVICE")
		return
	}

	// Create the compose config
	if err = repository.CreateServiceComposeConfig(r.Context(), tx, composeConfig); err != nil {
		zap.L().Error("Failed to create compose config", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to create compose config", "FAILED_TO_CREATE_COMPOSE_CONFIG")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Return the created service
	response.RespondWithJSON(w, http.StatusCreated, "Service created successfully", service)
}
