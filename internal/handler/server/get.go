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
// | Get Servers                                  |
// +----------------------------------------------+

// GetServers godoc
// @Summary Get all servers for a team
// @Description Gets all server configurations within a team that the user has access to
// @Tags server
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Success 200 {object} response.SuccessResponse{data=[]models.Server} "Servers retrieved successfully"
// @Failure 400 {object} response.ErrorResponse "Team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/servers [get]
// @Security BearerAuth
func (h *ServerHandler) GetServers(w http.ResponseWriter, r *http.Request) {
	// Get the team ID from the URL parameter
	teamID := chi.URLParam(r, "teamID")

	// Get the user ID from the context
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

	// Get the servers from the database
	servers, err := repository.GetServersByTeamID(r.Context(), tx, teamID)
	if err != nil {
		zap.L().Error("Failed to get servers", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get servers", "FAILED_TO_GET_SERVERS")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Return the servers
	response.RespondWithJSON(w, http.StatusOK, servers)
}

// +----------------------------------------------+
// | Get Server                                   |
// +----------------------------------------------+

// GetServer godoc
// @Summary Get a specific server
// @Description Gets a specific server configuration by ID within a team that the user has access to
// @Tags server
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param serverID path string true "Server ID"
// @Success 200 {object} response.SuccessResponse{data=models.Server} "Server retrieved successfully"
// @Failure 400 {object} response.ErrorResponse "Team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 404 {object} response.ErrorResponse "Server not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/servers/{serverID} [get]
// @Security BearerAuth
func (h *ServerHandler) GetServer(w http.ResponseWriter, r *http.Request) {
	// Get the team ID from the URL parameter
	teamID := chi.URLParam(r, "teamID")

	// Get the server ID from the URL parameter
	serverID := chi.URLParam(r, "serverID")

	// Get the user ID from the context
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

	// Get the server from the database
	server, err := repository.GetServerByID(r.Context(), tx, serverID, teamID)
	if err != nil {
		zap.L().Error("Failed to get server", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get server", "FAILED_TO_GET_SERVER")
		return
	}

	// Check if server was found
	if server == nil {
		response.RespondWithError(w, http.StatusNotFound, "Server not found", "SERVER_NOT_FOUND")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Return the server
	response.RespondWithJSON(w, http.StatusOK, server)
}
