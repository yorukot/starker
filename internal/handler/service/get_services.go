// +----------------------------------------------+
// | Get Services                                 |
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

// GetServices godoc
// @Summary Get all services for a project
// @Description Retrieves all services belonging to a specific project within a team
// @Tags service
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param projectID path string true "Project ID"
// @Success 200 {array} models.Service "List of services"
// @Failure 400 {object} response.ErrorResponse "Team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/projects/{projectID}/services [get]
// @Security BearerAuth
func (h *ServiceHandler) GetServices(w http.ResponseWriter, r *http.Request) {
	// Get URL parameters
	teamID := chi.URLParam(r, "teamID")
	projectID := chi.URLParam(r, "projectID")

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

	// Get services
	services, err := repository.GetServices(r.Context(), *h.Tx, teamID, projectID)
	if err != nil {
		zap.L().Error("Failed to get services", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get services", "FAILED_TO_GET_SERVICES")
		return
	}

	// Commit transaction
	repository.CommitTransaction(tx, r.Context())

	// Return services
	response.RespondWithJSON(w, http.StatusOK, services)
}
