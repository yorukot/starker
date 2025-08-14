package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/internal/service/serversvc"
	"github.com/yorukot/starker/pkg/response"
)

// +----------------------------------------------+
// | Update Server                                |
// +----------------------------------------------+

// UpdateServer godoc
// @Summary Update a server
// @Description Updates an existing server configuration within a team
// @Tags server
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param serverID path string true "Server ID"
// @Param request body serversvc.UpdateServerRequest true "Server update request"
// @Success 200 {object} response.SuccessResponse{data=models.Server} "Server updated successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body or team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 404 {object} response.ErrorResponse "Server not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/servers/{serverID} [put]
// @Security BearerAuth
func (h *ServerHandler) UpdateServer(w http.ResponseWriter, r *http.Request) {
	// Get the team ID and server ID from the URL parameters
	teamID := chi.URLParam(r, "teamID")
	serverID := chi.URLParam(r, "serverID")

	// Parse the request body into the update server request struct
	var updateServerRequest serversvc.UpdateServerRequest
	if err := json.NewDecoder(r.Body).Decode(&updateServerRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	// Validate the server update request
	if err := serversvc.ServerUpdateValidate(updateServerRequest); err != nil {
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

	// If private key ID is being updated, verify the new private key exists and belongs to the team
	if updateServerRequest.PrivateKeyID != nil {
		_, err = repository.GetPrivateKeyByID(r.Context(), tx, *updateServerRequest.PrivateKeyID, teamID)
		if err != nil {
			zap.L().Error("Failed to verify private key", zap.Error(err))
			response.RespondWithError(w, http.StatusBadRequest, "Private key not found or access denied", "PRIVATE_KEY_NOT_FOUND")
			return
		}
	}

	// Update the server in the database
	server, err := repository.UpdateServer(r.Context(), tx, teamID, serverID, updateServerRequest)
	if err != nil {
		zap.L().Error("Failed to update server", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to update server", "SERVER_UPDATE_FAILED")
		return
	}

	// Check if server was found and updated
	if server == nil {
		response.RespondWithError(w, http.StatusNotFound, "Server not found", "SERVER_NOT_FOUND")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Return success response with the updated server
	response.RespondWithJSON(w, http.StatusOK, "Server updated successfully", server)
}
