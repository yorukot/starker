// +----------------------------------------------+
// | Get Service Compose                          |
// +----------------------------------------------+

package service

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/response"
)

// GetServiceCompose godoc
// @Summary Get service Docker Compose configuration
// @Description Retrieves the Docker Compose configuration for a specific service
// @Tags service
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param projectID path string true "Project ID"
// @Param serviceID path string true "Service ID"
// @Success 200 {object} response.SuccessResponse{data=models.ServiceComposeConfig} "Service compose configuration"
// @Failure 400 {object} response.ErrorResponse "Team access denied or compose config not found"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/projects/{projectID}/services/{serviceID}/compose [get]
// @Security BearerAuth
func (h *ServiceHandler) GetServiceCompose(w http.ResponseWriter, r *http.Request) {
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

	// Get compose configuration
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

	// Commit transaction
	repository.CommitTransaction(tx, r.Context())

	// Return compose config
	response.RespondWithJSON(w, http.StatusOK, composeConfig)
}
