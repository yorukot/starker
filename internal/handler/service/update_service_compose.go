// +----------------------------------------------+
// | Update Service Compose                       |
// +----------------------------------------------+

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

// UpdateServiceCompose godoc
// @Summary Update service Docker Compose configuration
// @Description Updates the Docker Compose configuration for a specific service
// @Tags service
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param projectID path string true "Project ID"
// @Param serviceID path string true "Service ID"
// @Param request body servicesvc.UpdateServiceComposeRequest true "Service compose update request"
// @Success 200 {object} response.SuccessResponse{data=models.ServiceComposeConfig} "Compose config updated successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body, team access denied, or service not found"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/projects/{projectID}/services/{serviceID}/compose [patch]
// @Security BearerAuth
func (h *ServiceHandler) UpdateServiceCompose(w http.ResponseWriter, r *http.Request) {
	// Get URL parameters
	teamID := chi.URLParam(r, "teamID")
	projectID := chi.URLParam(r, "projectID")
	serviceID := chi.URLParam(r, "serviceID")

	// Decode the request body
	var updateServiceComposeRequest servicesvc.UpdateServiceComposeRequest
	if err := json.NewDecoder(r.Body).Decode(&updateServiceComposeRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	// Validate the request body
	if err := servicesvc.ServiceComposeUpdateValidate(updateServiceComposeRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	// Get user ID from context
	userID := r.Context().Value(middleware.UserIDKey).(string)

	// Start database transaction
	tx, err := repository.StartTransaction(h.DB, r.Context())
	if err != nil {
		zap.L().Error("Failed to begin transaction", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
		return
	}
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

	// Verify service exists and user has access
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

	// Get existing compose configuration
	composeConfig, err := repository.GetServiceComposeConfig(r.Context(), tx, serviceID)
	if err != nil {
		zap.L().Error("Failed to get compose config", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get compose config", "FAILED_TO_GET_COMPOSE_CONFIG")
		return
	}
	if composeConfig == nil {
		response.RespondWithError(w, http.StatusBadRequest, "Compose config not found", "COMPOSE_CONFIG_NOT_FOUND")
		return
	}

	// Update compose config with new values
	updatedComposeConfig := servicesvc.UpdateServiceComposeFromRequest(*composeConfig, updateServiceComposeRequest)

	// Update the compose config in database
	if err := repository.UpdateServiceComposeConfig(r.Context(), tx, updatedComposeConfig); err != nil {
		zap.L().Error("Failed to update compose config", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to update compose config", "FAILED_TO_UPDATE_COMPOSE_CONFIG")
		return
	}

	// Commit transaction
	repository.CommitTransaction(tx, r.Context())

	// Return the updated compose config
	response.RespondWithJSON(w, http.StatusOK, updatedComposeConfig)
}
