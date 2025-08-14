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
// | Create Server                                |
// +----------------------------------------------+

// CreateServer godoc
// @Summary Create a new server
// @Description Creates a new server configuration for SSH connections within a team. Tests the connection before saving.
// @Tags server
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param request body serversvc.CreateServerRequest true "Server creation request"
// @Success 201 {object} response.SuccessResponse{data=models.Server} "Server created successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body, team access denied, or server connection failed"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/servers [post]
// @Security BearerAuth
func (h *ServerHandler) CreateServer(w http.ResponseWriter, r *http.Request) {
	// Get the team ID from the URL parameter
	teamID := chi.URLParam(r, "teamID")

	// Get the server from the request body
	var createServerRequest serversvc.CreateServerRequest
	if err := json.NewDecoder(r.Body).Decode(&createServerRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	// Validate the server creation request
	if err := serversvc.ServerValidate(createServerRequest); err != nil {
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

	// Verify the private key exists and belongs to the team
	privateKey, err := repository.GetPrivateKeyByID(r.Context(), tx, createServerRequest.PrivateKeyID, teamID)
	if err != nil {
		zap.L().Error("Failed to verify private key", zap.Error(err))
		response.RespondWithError(w, http.StatusBadRequest, "Private key not found or access denied", "PRIVATE_KEY_NOT_FOUND")
		return
	}

	// Generate the server
	server := serversvc.GenerateServer(createServerRequest, teamID)

	// Test the server connection before creating it
	if err = serversvc.TestServerConnection(r.Context(), server, *privateKey, h.SSHPool); err != nil {
		zap.L().Error("Failed to test server connection", zap.Error(err))
		response.RespondWithError(w, http.StatusBadRequest, "Failed to connect to server with provided credentials", "SERVER_CONNECTION_FAILED")
		return
	}

	// Create the server
	if err = repository.CreateServer(r.Context(), tx, server); err != nil {
		zap.L().Error("Failed to create server", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to create server", "FAILED_TO_CREATE_SERVER")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Return the created server
	response.RespondWithJSON(w, http.StatusCreated, "Server created successfully", server)
}
