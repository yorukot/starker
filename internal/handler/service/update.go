package service

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/response"
)

type updateServiceRequest struct {
	Name        *string              `json:"name,omitempty" validate:"omitempty,min=3,max=255"`
	Description *string              `json:"description,omitempty" validate:"omitempty,max=500"`
	Type        *string              `json:"type,omitempty" validate:"omitempty,oneof=docker compose"`
	State       *models.ServiceState `json:"status,omitempty" validate:"omitempty,oneof=running stopped starting stopping"`
}

// +----------------------------------------------+
// | Update Service                               |
// +----------------------------------------------+

// UpdateService godoc
// @Summary Update service metadata
// @Description Updates service metadata (name, description, type) within a team and project
// @Tags service
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param projectID path string true "Project ID"
// @Param serviceID path string true "Service ID"
// @Param request body servicesvc.UpdateServiceRequest true "Service update request"
// @Success 200 {object} response.SuccessResponse{data=models.Service} "Service updated successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body or team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 404 {object} response.ErrorResponse "Service not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/projects/{projectID}/services/{serviceID} [patch]
// @Security BearerAuth
func (h *ServiceHandler) UpdateService(w http.ResponseWriter, r *http.Request) {
	// Get teamID, projectID and serviceID from the request
	teamID := chi.URLParam(r, "teamID")
	projectID := chi.URLParam(r, "projectID")
	serviceID := chi.URLParam(r, "serviceID")

	// Decode the request body
	var updateServiceRequest updateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&updateServiceRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	// Validate the request body
	if err := validator.New().Struct(updateServiceRequest); err != nil {
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

	// Check if the service exists
	service, err := repository.GetServiceByID(r.Context(), tx, serviceID, teamID, projectID)
	if err != nil {
		zap.L().Error("Failed to find service", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to find service", "FAILED_TO_FIND_SERVICE")
		return
	}

	if service == nil {
		response.RespondWithError(w, http.StatusBadRequest, "Service not found", "SERVICE_NOT_FOUND")
		return
	}

	// Update service fields if provided
	updatedService := updateServiceFromRequest(*service, updateServiceRequest)

	// Update the service in database
	if err := repository.UpdateService(r.Context(), tx, updatedService); err != nil {
		zap.L().Error("Failed to update service", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to update service", "FAILED_TO_UPDATE_SERVICE")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Return the updated service
	response.RespondWithJSON(w, http.StatusOK, updatedService)
}

// updateServiceFromRequest updates a service model with new values from update request
func updateServiceFromRequest(existingService models.Service, updateServiceRequest updateServiceRequest) models.Service {
	if updateServiceRequest.Name != nil {
		existingService.Name = *updateServiceRequest.Name
	}
	if updateServiceRequest.Description != nil {
		existingService.Description = updateServiceRequest.Description
	}
	if updateServiceRequest.Type != nil {
		existingService.Type = *updateServiceRequest.Type
	}
	if updateServiceRequest.State != nil {
		existingService.State = *updateServiceRequest.State
	}
	existingService.UpdatedAt = time.Now()

	return existingService
}
