// +----------------------------------------------+
// | Update Service Status                        |
// +----------------------------------------------+

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/core"
	"github.com/yorukot/starker/internal/core/dockerutils"
	"github.com/yorukot/starker/internal/handler/service/utils"
	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/dockeryaml"
	"github.com/yorukot/starker/pkg/generator"
	"github.com/yorukot/starker/pkg/response"
)

// updateServiceStateRequest represents a request to update service state
type updateServiceStateRequest struct {
	State string `json:"state" validate:"required,oneof=start stop restart rebuild" example:"start"` // Service state action (start, stop, restart, rebuild)
}

// UpdateServiceState godoc
// @Summary Update service state with SSE streaming
// @Description Updates service state (start/stop/restart/rebuild) with real-time progress streaming via Server-Sent Events
// @Tags service
// @Accept json
// @Produce text/event-stream
// @Param teamID path string true "Team ID"
// @Param projectID path string true "Project ID"
// @Param serviceID path string true "Service ID"
// @Param request body updateServiceStateRequest true "Service state update request"
// @Success 200 {string} string "SSE stream of service state updates"
// @Failure 400 {object} response.ErrorResponse "Invalid request body or team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 404 {object} response.ErrorResponse "Service not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/projects/{projectID}/services/{serviceID}/state [patch]
// @Security BearerAuth
func (h *ServiceHandler) UpdateServiceState(w http.ResponseWriter, r *http.Request) {
	// Get URL parameters
	teamID := chi.URLParam(r, "teamID")
	projectID := chi.URLParam(r, "projectID")
	serviceID := chi.URLParam(r, "serviceID")

	// Decode and validate request body
	var updateServiceStateRequest updateServiceStateRequest
	if err := json.NewDecoder(r.Body).Decode(&updateServiceStateRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	if err := validator.New().Struct(updateServiceStateRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	newState := updateServiceStateRequest.State

	// Get userID
	userID := r.Context().Value(middleware.UserIDKey).(string)

	// Start database transaction
	tx, err := repository.StartTransaction(h.DB, r.Context())
	if err != nil {
		zap.L().Error("Failed to begin transaction", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
		return
	}
	defer repository.DeferRollback(tx, r.Context())

	// Verify user has access to the team and service exists in the project
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
		response.RespondWithError(w, http.StatusBadRequest, "Service not found", "SERVICE_NOT_FOUND")
		return
	}

	// Check if the current state allows the requested operation
	if !checkStateIsRight(service.State, newState) {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid state transition", "INVALID_STATE_TRANSITION")
		return
	}

	// Execute the service operation
	result, err := h.executeServiceOperation(r.Context(), tx, newState, service)
	if err != nil {
		zap.L().Error("Failed to execute service command", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to execute service command", "FAILED_TO_EXECUTE_COMMAND")
		return
	}

	if err := repository.UpdateService(r.Context(), tx, *service); err != nil {
		zap.L().Error("Failed to update initial service status", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to update service status", "FAILED_TO_UPDATE_SERVICE_STATUS")
		return
	}

	// Stream the operation with real-time updates
	utils.StreamServiceOutputWithUpdate(r.Context(), w, result, service, &tx, newState)
}

func checkStateIsRight(state models.ServiceState, newState string) bool {
	switch newState {
	case "start":
		return state == models.ServiceStateStopped
	case "stop":
		return state == models.ServiceStateRunning
	case "restart":
		return state == models.ServiceStateRunning
	case "rebuild":
		return state == models.ServiceStateRunning || state == models.ServiceStateStopped
	default:
		return false
	}
}

// executeServiceOperation executes the Docker service operation and returns streaming result
func (h *ServiceHandler) executeServiceOperation(ctx context.Context, tx pgx.Tx, operation string, service *models.Service) (*core.StreamChan, error) {
	dockerHandler, streamChan, err := h.setupDockerHandler(ctx, tx, service)
	if err != nil {
		return nil, err
	}

	go func() {
		switch operation {
		case "start":
			dockerHandler.StartDockerCompose(context.Background())
		case "stop":
			dockerHandler.StopDockerCompose(context.Background())
		case "restart":
			dockerHandler.RestartDockerCompose(context.Background())
		case "rebuild":
			dockerHandler.RebuildDockerCompose(context.Background())
		}
	}()
	return streamChan, nil
}

// setupDockerHandler handles the common setup logic for Docker operations
func (h *ServiceHandler) setupDockerHandler(ctx context.Context, tx pgx.Tx, service *models.Service) (*dockerutils.DockerHandler, *core.StreamChan, error) {
	// Get the service compose configuration
	composeConfig, err := repository.GetServiceComposeConfig(ctx, tx, service.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get service compose config: %w", err)
	}
	if composeConfig == nil {
		return nil, nil, fmt.Errorf("no compose configuration found for service")
	}

	// Get server details for connection
	server, err := repository.GetServerByID(ctx, tx, service.ServerID, service.TeamID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get server: %w", err)
	}
	if server == nil {
		return nil, nil, fmt.Errorf("server not found")
	}

	// Get private key for SSH connection
	privateKey, err := repository.GetPrivateKeyByID(ctx, tx, server.PrivateKeyID, service.TeamID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get private key: %w", err)
	}
	if privateKey == nil {
		return nil, nil, fmt.Errorf("private key not found")
	}

	// Parse the Docker Compose configuration
	namingGenerator := generator.NewNamingGenerator(service.ID, service.TeamID, service.ServerID, service.ProjectID)
	project, err := dockeryaml.ParseComposeContent(composeConfig.ComposeFile, namingGenerator.ProjectName())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	// Validate the compose project
	if err := dockeryaml.Validate(project); err != nil {
		return nil, nil, fmt.Errorf("invalid compose configuration: %w", err)
	}

	// Get Docker client from connection pool
	connectionID := namingGenerator.ConnectionID()
	// Build SSH connection string
	sshClient, err := h.ConnectionPool.GetSSHConnection(connectionID, server.Host, server.Port, server.User, []byte(privateKey.PrivateKey))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get Docker connection: %w", err)
	}

	// Create streaming channels
	streamChan := core.NewStreamChan()

	// Create Docker handler
	dockerHandler := &dockerutils.DockerHandler{
		Client:          sshClient,
		Project:         project,
		NamingGenerator: namingGenerator,
		DB:              h.DB,
		ConnectionPool:  h.ConnectionPool,
		StreamChan:      streamChan,
	}

	return dockerHandler, &streamChan, nil
}
