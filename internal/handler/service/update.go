package service

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/internal/service/servicesvc"
	"github.com/yorukot/starker/pkg/response"
)

// +----------------------------------------------+
// | Update Service Status                        |
// +----------------------------------------------+

// UpdateService godoc
// @Summary Update a service status
// @Description Updates a service status (start, stop, restart) within a team and project
// @Tags service
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param projectID path string true "Project ID"
// @Param serviceID path string true "Service ID"
// @Param request body servicesvc.UpdateServiceRequest true "Service update request"
// @Success 200 {object} response.SuccessResponse{data=models.Service} "Service updated successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body, team access denied, or invalid status transition"
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
	var updateServiceRequest servicesvc.UpdateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&updateServiceRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	// Validate the request body
	if err := servicesvc.ServiceUpdateValidate(updateServiceRequest); err != nil {
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

	// Get the service
	service, err := repository.GetServiceByID(r.Context(), tx, serviceID, teamID, projectID)
	if err != nil {
		zap.L().Error("Failed to get service", zap.Error(err))
		response.RespondWithError(w, http.StatusNotFound, "Service not found", "SERVICE_NOT_FOUND")
		return
	}

	// Handle status changes if requested
	if updateServiceRequest.Status != nil {
		newStatus := *updateServiceRequest.Status
		currentStatus := service.Status

		switch newStatus {
		case models.ServiceStatusStarting:
			if currentStatus == models.ServiceStatusStopped {
				// Update status to "starting" first
				service.Status = models.ServiceStatusStarting
				service.UpdatedAt = time.Now()
				if err := repository.UpdateService(r.Context(), tx, *service); err != nil {
					zap.L().Error("Failed to update service status to starting", zap.Error(err))
					response.RespondWithError(w, http.StatusInternalServerError, "Failed to update service status", "SERVICE_UPDATE_FAILED")
					return
				}

				// Commit the transaction before long-running operation
				repository.CommitTransaction(tx, r.Context())

				// Start a new transaction for final status update
				tx, err = repository.StartTransaction(h.DB, r.Context())
				if err != nil {
					zap.L().Error("Failed to begin final transaction", zap.Error(err))
					response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
					return
				}
				defer repository.DeferRollback(tx, r.Context())

				// Start the service
				if err := StartService(r.Context(), serviceID, teamID, projectID, tx, h.SSHPool); err != nil {
					zap.L().Error("Failed to start service", zap.Error(err))
					// Update status back to stopped on failure
					service.Status = models.ServiceStatusStopped
					service.UpdatedAt = time.Now()
					repository.UpdateService(r.Context(), tx, *service)
					response.RespondWithError(w, http.StatusInternalServerError, "Failed to start service", "SERVICE_START_FAILED")
					return
				}

				// Update status to running
				service.Status = models.ServiceStatusRunning
				service.LastDeployedAt = &time.Time{}
				*service.LastDeployedAt = time.Now()
			} else if currentStatus == models.ServiceStatusRunning {
				// Restart: stop then start
				service.Status = models.ServiceStatusStopping
				service.UpdatedAt = time.Now()
				if err := repository.UpdateService(r.Context(), tx, *service); err != nil {
					zap.L().Error("Failed to update service status to stopping", zap.Error(err))
					response.RespondWithError(w, http.StatusInternalServerError, "Failed to update service status", "SERVICE_UPDATE_FAILED")
					return
				}

				// Commit before long operation
				repository.CommitTransaction(tx, r.Context())

				// Start new transaction
				tx, err = repository.StartTransaction(h.DB, r.Context())
				if err != nil {
					zap.L().Error("Failed to begin transaction", zap.Error(err))
					response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
					return
				}
				defer repository.DeferRollback(tx, r.Context())

				// Stop the service
				if err := StopService(r.Context(), serviceID, teamID, projectID, tx, h.SSHPool); err != nil {
					zap.L().Error("Failed to stop service for restart", zap.Error(err))
					response.RespondWithError(w, http.StatusInternalServerError, "Failed to stop service", "SERVICE_STOP_FAILED")
					return
				}

				// Update to starting
				service.Status = models.ServiceStatusStarting
				service.UpdatedAt = time.Now()
				if err := repository.UpdateService(r.Context(), tx, *service); err != nil {
					zap.L().Error("Failed to update service status to starting", zap.Error(err))
					response.RespondWithError(w, http.StatusInternalServerError, "Failed to update service status", "SERVICE_UPDATE_FAILED")
					return
				}

				// Commit before starting
				repository.CommitTransaction(tx, r.Context())

				// Start new transaction
				tx, err = repository.StartTransaction(h.DB, r.Context())
				if err != nil {
					zap.L().Error("Failed to begin transaction", zap.Error(err))
					response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
					return
				}
				defer repository.DeferRollback(tx, r.Context())

				// Start the service
				if err := StartService(r.Context(), serviceID, teamID, projectID, tx, h.SSHPool); err != nil {
					zap.L().Error("Failed to restart service", zap.Error(err))
					service.Status = models.ServiceStatusStopped
					service.UpdatedAt = time.Now()
					repository.UpdateService(r.Context(), tx, *service)
					response.RespondWithError(w, http.StatusInternalServerError, "Failed to restart service", "SERVICE_RESTART_FAILED")
					return
				}

				// Update status to running
				service.Status = models.ServiceStatusRunning
				service.LastDeployedAt = &time.Time{}
				*service.LastDeployedAt = time.Now()
			} else {
				response.RespondWithError(w, http.StatusBadRequest, "Cannot start service in current state", "INVALID_STATUS_TRANSITION")
				return
			}

		case models.ServiceStatusStopping:
			if currentStatus == models.ServiceStatusRunning {
				// Update status to "stopping" first
				service.Status = models.ServiceStatusStopping
				service.UpdatedAt = time.Now()
				if err := repository.UpdateService(r.Context(), tx, *service); err != nil {
					zap.L().Error("Failed to update service status to stopping", zap.Error(err))
					response.RespondWithError(w, http.StatusInternalServerError, "Failed to update service status", "SERVICE_UPDATE_FAILED")
					return
				}

				// Commit before long operation
				repository.CommitTransaction(tx, r.Context())

				// Start new transaction
				tx, err = repository.StartTransaction(h.DB, r.Context())
				if err != nil {
					zap.L().Error("Failed to begin transaction", zap.Error(err))
					response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
					return
				}
				defer repository.DeferRollback(tx, r.Context())

				// Stop the service
				if err := StopService(r.Context(), serviceID, teamID, projectID, tx, h.SSHPool); err != nil {
					zap.L().Error("Failed to stop service", zap.Error(err))
					service.Status = models.ServiceStatusRunning
					service.UpdatedAt = time.Now()
					repository.UpdateService(r.Context(), tx, *service)
					response.RespondWithError(w, http.StatusInternalServerError, "Failed to stop service", "SERVICE_STOP_FAILED")
					return
				}

				// Update status to stopped
				service.Status = models.ServiceStatusStopped
			} else {
				response.RespondWithError(w, http.StatusBadRequest, "Cannot stop service in current state", "INVALID_STATUS_TRANSITION")
				return
			}

		default:
			response.RespondWithError(w, http.StatusBadRequest, "Invalid status transition", "INVALID_STATUS_TRANSITION")
			return
		}
	}

	// Apply other updates from the request
	updatedService := servicesvc.UpdateServiceFromRequest(*service, updateServiceRequest)

	// Update the service in the database
	if err := repository.UpdateService(r.Context(), tx, updatedService); err != nil {
		zap.L().Error("Failed to update service", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to update service", "SERVICE_UPDATE_FAILED")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Return the updated service
	response.RespondWithJSON(w, http.StatusOK, "Service updated successfully", updatedService)
}
