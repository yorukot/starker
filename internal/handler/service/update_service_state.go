// +----------------------------------------------+
// | Update Service Status                        |
// +----------------------------------------------+

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/core/dockerutils"
	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/dockeryaml"
	"github.com/yorukot/starker/pkg/generator"
	"github.com/yorukot/starker/pkg/response"
)

// updateServiceStateRequest represents a request to update service state
type updateServiceStateRequest struct {
	State string `json:"state" validate:"required,oneof=start stop restart" example:"start"` // Service state action (start, stop, restart)
}

// UpdateServiceState godoc
// @Summary Update service state with SSE streaming
// @Description Updates service state (start/stop/restart) with real-time progress streaming via Server-Sent Events
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
	h.Tx = &tx
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
	service, err := repository.GetServiceByID(r.Context(), *h.Tx, serviceID, teamID, projectID)
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
	result, err := h.executeServiceOperation(r.Context(), newState, service)
	if err != nil {
		zap.L().Error("Failed to execute service command", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to execute service command", "FAILED_TO_EXECUTE_COMMAND")
		return
	}

	// Update service to initial status before streaming
	service.State = models.ServiceStateStarting
	if err := repository.UpdateService(r.Context(), *h.Tx, *service); err != nil {
		zap.L().Error("Failed to update initial service status", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to update service status", "FAILED_TO_UPDATE_SERVICE_STATUS")
		return
	}

	// Stream the operation with real-time updates
	h.streamServiceOutputWithUpdate(r.Context(), w, result, service)
}

func checkStateIsRight(state models.ServiceState, newState string) bool {
	switch newState {
	case "start":
		return state == models.ServiceStateStopped
	case "stop":
		return state == models.ServiceStateRunning
	case "restart":
		return state == models.ServiceStateRunning
	default:
		return false
	}
}

// executeServiceOperation executes the Docker service operation and returns streaming result
func (h *ServiceHandler) executeServiceOperation(ctx context.Context, operation string, service *models.Service) (*dockerutils.StreamingResult, error) {
	switch operation {
	case "start":
		return h.executeStartOperation(ctx, service)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// executeStartOperation handles the Docker compose start operation
func (h *ServiceHandler) executeStartOperation(ctx context.Context, service *models.Service) (*dockerutils.StreamingResult, error) {
	// Get the service compose configuration
	composeConfig, err := repository.GetServiceComposeConfig(ctx, *h.Tx, service.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get service compose config: %w", err)
	}
	if composeConfig == nil {
		return nil, fmt.Errorf("no compose configuration found for service")
	}

	// Get server details for connection
	server, err := repository.GetServerByID(ctx, *h.Tx, service.ServerID, service.TeamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}
	if server == nil {
		return nil, fmt.Errorf("server not found")
	}

	// Get private key for SSH connection
	privateKey, err := repository.GetPrivateKeyByID(ctx, *h.Tx, server.PrivateKeyID, service.TeamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get private key: %w", err)
	}
	if privateKey == nil {
		return nil, fmt.Errorf("private key not found")
	}

	// Parse the Docker Compose configuration
	namingGenerator := generator.NewNamingGenerator(service.ID, service.TeamID, service.ServerID)
	project, err := dockeryaml.ParseComposeContent(composeConfig.ComposeFile, namingGenerator.ProjectName())
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	// Validate the compose project
	if err := dockeryaml.Validate(project); err != nil {
		return nil, fmt.Errorf("invalid compose configuration: %w", err)
	}

	// Get Docker client from connection pool
	connectionID := namingGenerator.ConnectionID()
	// Build SSH connection string
	sshHost := fmt.Sprintf("%s@%s:%s", server.User, server.IP, server.Port)
	dockerClient, err := h.ConnectionPool.GetDockerConnection(connectionID, sshHost, []byte(privateKey.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to get Docker connection: %w", err)
	}

	// Create streaming channels
	streamChan := dockerutils.NewStreamChan()

	// Create Docker handler
	dockerHandler := &dockerutils.DockerHandler{
		Client:          dockerClient,
		Project:         project,
		NamingGenerator: namingGenerator,
		DB:              h.DB,
		ConnectionPool:  h.ConnectionPool,
		StreamChan:      streamChan,
	}

	// Start the Docker compose operation
	err = dockerHandler.StartDockerCompose(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start Docker compose: %w", err)
	}

	// Return the streaming result
	return &dockerutils.StreamingResult{
		StreamChan: streamChan,
	}, nil
}

// streamServiceOutputWithUpdate handles SSE streaming of Docker operation progress
func (h *ServiceHandler) streamServiceOutputWithUpdate(ctx context.Context, w http.ResponseWriter, result *dockerutils.StreamingResult, service *models.Service) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Get flusher for real-time streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		zap.L().Error("Streaming unsupported")
		response.RespondWithError(w, http.StatusInternalServerError, "Streaming unsupported", "STREAMING_UNSUPPORTED")
		return
	}

	// Send initial event
	data, _ := json.Marshal(map[string]interface{}{
		"message": "Starting Docker service",
		"type":    "info",
	})
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()

	// Stream the operation progress
	for {
		select {
		case <-ctx.Done():
			zap.L().Info("Client disconnected")
			return

		case logMsg := <-result.StreamChan.LogChan:
			// Stream log message
			data, _ := json.Marshal(map[string]interface{}{
				"message": logMsg.Message,
				"type":    string(logMsg.Type),
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

		case errMsg := <-result.StreamChan.ErrChan:
			// Stream error message
			data, _ := json.Marshal(map[string]interface{}{
				"message": errMsg.Message,
				"type":    string(errMsg.Type),
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

		case finalErr := <-result.StreamChan.FinalError:
			// Operation failed - rollback transaction
			zap.L().Error("Docker operation failed", zap.Error(finalErr))

			// Update service state to stopped (rollback)
			service.State = models.ServiceStateStopped
			if updateErr := repository.UpdateService(ctx, *h.Tx, *service); updateErr != nil {
				zap.L().Error("Failed to rollback service state", zap.Error(updateErr))
			}

			data, _ := json.Marshal(map[string]interface{}{
				"message": fmt.Sprintf("Operation failed: %v", finalErr),
				"type":    "error",
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			return

		case <-result.StreamChan.DoneChan:
			// Operation completed successfully
			zap.L().Info("Docker operation completed successfully")

			// Update service state to running
			service.State = models.ServiceStateRunning
			service.LastDeployedAt = &[]time.Time{time.Now()}[0]
			if err := repository.UpdateService(ctx, *h.Tx, *service); err != nil {
				zap.L().Error("Failed to update service state", zap.Error(err))
				data, _ := json.Marshal(map[string]interface{}{
					"message": "Failed to update service state in database",
					"type":    "error",
				})
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
				return
			}

			// Commit the transaction
			if err := (*h.Tx).Commit(ctx); err != nil {
				zap.L().Error("Failed to commit transaction", zap.Error(err))
				data, _ := json.Marshal(map[string]interface{}{
					"message": "Failed to commit database transaction",
					"type":    "error",
				})
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
				return
			}

			// Send success completion event
			data, _ := json.Marshal(map[string]interface{}{
				"message": "Service started successfully",
				"type":    "info",
				"state":   "running",
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			return
		}
	}
}
