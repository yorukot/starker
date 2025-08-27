package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/response"
)

// +----------------------------------------------+
// | Delete Server                                |
// +----------------------------------------------+

// DeleteServer godoc
// @Summary Delete a server
// @Description Deletes a specific server configuration by ID within a team
// @Tags server
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param serverID path string true "Server ID"
// @Success 200 {object} response.SuccessResponse "Server deleted successfully"
// @Failure 400 {object} response.ErrorResponse "Team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 404 {object} response.ErrorResponse "Server not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/servers/{serverID} [delete]
// @Security BearerAuth
func (h *ServerHandler) DeleteServer(w http.ResponseWriter, r *http.Request) {
	// Get the team ID and server ID from the URL
	teamID := chi.URLParam(r, "teamID")
	serverID := chi.URLParam(r, "serverID")

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

	// Check if the server exists before deleting
	server, err := repository.GetServerByID(r.Context(), tx, serverID, teamID)
	if err != nil {
		zap.L().Error("Failed to get server", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get server", "FAILED_TO_GET_SERVER")
		return
	}

	if server == nil {
		response.RespondWithError(w, http.StatusNotFound, "Server not found", "SERVER_NOT_FOUND")
		return
	}

	// Delete the server
	if err = repository.DeleteServerByID(r.Context(), tx, serverID, teamID); err != nil {
		zap.L().Error("Failed to delete server", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to delete server", "FAILED_TO_DELETE_SERVER")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Response
	response.RespondWithJSON(w, http.StatusOK, nil)
}
