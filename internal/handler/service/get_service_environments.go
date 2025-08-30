package service

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/response"
)

// +----------------------------------------------+
// | Get Service Environments                     |
// +----------------------------------------------+

// GetServiceEnvironments godoc
// @Summary Get all environment variables for a service
// @Description Retrieves all environment variables for a specific service within a team and project
// @Tags service
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param projectID path string true "Project ID"
// @Param serviceID path string true "Service ID"
// @Success 200 {object} response.SuccessResponse{data=[]models.ServiceEnvironment} "Environment variables retrieved successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request or team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 404 {object} response.ErrorResponse "Service not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/projects/{projectID}/services/{serviceID}/env [get]
// @Security BearerAuth
func (h *ServiceHandler) GetServiceEnvironments(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamID")
	projectID := chi.URLParam(r, "projectID")
	serviceID := chi.URLParam(r, "serviceID")

	userID := r.Context().Value(middleware.UserIDKey).(string)

	tx, err := repository.StartTransaction(h.DB, r.Context())
	if err != nil {
		zap.L().Error("Failed to begin transaction", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
		return
	}
	defer repository.DeferRollback(tx, r.Context())

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

	environments, err := repository.GetServiceEnvironments(r.Context(), tx, serviceID)
	if err != nil {
		zap.L().Error("Failed to get service environments", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get service environments", "FAILED_TO_GET_SERVICE_ENVIRONMENTS")
		return
	}

	repository.CommitTransaction(tx, r.Context())
	response.RespondWithJSON(w, http.StatusOK, environments)
}
