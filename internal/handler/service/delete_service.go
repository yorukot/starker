package service

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/response"
)

// DeleteService godoc
// @Summary Delete service with complete cleanup
// @Description Deletes a service including stopping Docker containers, removing networks/volumes, and cleaning up all related database records
// @Tags service
// @Produce json
// @Param teamID path string true "Team ID"
// @Param projectID path string true "Project ID"
// @Param serviceID path string true "Service ID"
// @Success 200 {object} response.SuccessResponse "Service deleted successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request or team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 404 {object} response.ErrorResponse "Service not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/projects/{projectID}/services/{serviceID} [delete]
// @Security BearerAuth
func (h *ServiceHandler) DeleteService(w http.ResponseWriter, r *http.Request) {
	// Get URL parameters
	teamID := chi.URLParam(r, "teamID")
	projectID := chi.URLParam(r, "projectID")
	serviceID := chi.URLParam(r, "serviceID")

	// Get userID
	userID := r.Context().Value(middleware.UserIDKey).(string)

	// Start database transaction
	tx, err := repository.StartTransaction(h.DB, r.Context())
	if err != nil {
		zap.L().Error("Failed to begin transaction", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
		return
	}
	h.Tx = &tx
	defer repository.DeferRollback(tx, r.Context())

	// Verify user has access to the team
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

	// Check if service exists
	service, err := repository.GetServiceByID(r.Context(), tx, serviceID, teamID, projectID)
	if err != nil {
		zap.L().Error("Failed to find service", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to find service", "FAILED_TO_FIND_SERVICE")
		return
	}
	if service == nil {
		response.RespondWithError(w, http.StatusNotFound, "Service not found", "SERVICE_NOT_FOUND")
		return
	}

	// Check if the service still running
	if service.State == models.ServiceStateRunning || service.State == models.ServiceStateStarting {
		response.RespondWithError(w, http.StatusBadRequest, "You need to stop the service before deleting it", "FAILED_TO_DELETE_SERVICE")
		return
	}

	// Delete all related service data
	err = h.deleteAllServiceData(r.Context(), tx, serviceID)
	if err != nil {
		zap.L().Error("Failed to delete service data", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to delete service data", "FAILED_TO_DELETE_SERVICE_DATA")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	response.RespondWithJSON(w, http.StatusOK, nil)
}

// deleteAllServiceData deletes all service-related data from the database
func (h *ServiceHandler) deleteAllServiceData(ctx context.Context, tx pgx.Tx, serviceID string) error {
	// Delete in proper order due to foreign key constraints
	// Start with dependent tables first, then work up to the main service record

	// Delete service containers
	if err := repository.DeleteServiceContainers(ctx, tx, serviceID); err != nil {
		return err
	}

	// Delete service networks
	if err := repository.DeleteServiceNetworks(ctx, tx, serviceID); err != nil {
		return err
	}

	// Delete service volumes
	if err := repository.DeleteServiceVolumes(ctx, tx, serviceID); err != nil {
		return err
	}

	// Delete service images
	if err := repository.DeleteServiceImages(ctx, tx, serviceID); err != nil {
		return err
	}

	// Delete service compose config
	if err := repository.DeleteServiceComposeConfig(ctx, tx, serviceID); err != nil {
		return err
	}

	// Delete service git source (if exists)
	if err := repository.DeleteServiceSourceGit(ctx, tx, serviceID); err != nil {
		return err
	}

	// Finally, delete the main service record
	// We need to get the service details first for proper deletion
	service, err := repository.GetServiceByID(ctx, tx, serviceID, "", "")
	if err != nil {
		return err
	}
	if service != nil {
		if err := repository.DeleteService(ctx, tx, serviceID, service.TeamID, service.ProjectID); err != nil {
			return err
		}
	}

	return nil
}
