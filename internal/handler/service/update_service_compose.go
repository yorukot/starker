// +----------------------------------------------+
// | Update Service Compose                       |
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

type updateServiceComposeRequest struct {
	ComposeFile     *string `json:"compose_file,omitempty" validate:"omitempty,required"`
	ComposeFilePath *string `json:"compose_file_path,omitempty" validate:"omitempty,max=500"`
}

// UpdateServiceCompose godoc
// @Summary Update service Docker Compose configuration
// @Description Updates the Docker Compose configuration for a specific service
// @Tags service
// @Accept json
// @Produce text/event-stream
// @Param teamID path string true "Team ID"
// @Param projectID path string true "Project ID"
// @Param serviceID path string true "Service ID"
// @Param request body updateServiceComposeRequest true "Service compose update request"
// @Success 200 {string} string "SSE stream of compose update and rebuild progress"
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
	var updateServiceComposeRequest updateServiceComposeRequest
	if err := json.NewDecoder(r.Body).Decode(&updateServiceComposeRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	// Validate the request body
	if err := validator.New().Struct(updateServiceComposeRequest); err != nil {
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
	updatedComposeConfig := updateServiceComposeFromRequest(*composeConfig, updateServiceComposeRequest)

	// Update the compose config in database
	if err := repository.UpdateServiceComposeConfig(r.Context(), tx, updatedComposeConfig); err != nil {
		zap.L().Error("Failed to update compose config", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to update compose config", "FAILED_TO_UPDATE_COMPOSE_CONFIG")
		return
	}

	// Set service state to rebuilding
	service.State = models.ServiceStateRebuilding
	if err := repository.UpdateService(r.Context(), tx, *service); err != nil {
		zap.L().Error("Failed to update service state to rebuilding", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to update service status", "FAILED_TO_UPDATE_SERVICE_STATUS")
		return
	}

	// Execute the rebuild operation
	result, err := h.executeRebuildOperationForComposeUpdate(r.Context(), tx, service)
	if err != nil {
		zap.L().Error("Failed to execute rebuild operation", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to execute rebuild operation", "FAILED_TO_EXECUTE_REBUILD")
		return
	}

	// Stream the rebuild operation with real-time updates
	utils.StreamServiceOutputWithUpdate(r.Context(), w, result, service, &tx, "rebuild")
}

// executeRebuildOperationForComposeUpdate handles the Docker compose rebuild operation for compose updates
func (h *ServiceHandler) executeRebuildOperationForComposeUpdate(ctx context.Context, tx pgx.Tx, service *models.Service) (*core.StreamChan, error) {
	// Get the service compose configuration (with updated content)
	composeConfig, err := repository.GetServiceComposeConfig(ctx, tx, service.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get service compose config: %w", err)
	}
	if composeConfig == nil {
		return nil, fmt.Errorf("no compose configuration found for service")
	}

	// Get server details for connection
	server, err := repository.GetServerByID(ctx, tx, service.ServerID, service.TeamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}
	if server == nil {
		return nil, fmt.Errorf("server not found")
	}

	// Get private key for SSH connection
	privateKey, err := repository.GetPrivateKeyByID(ctx, tx, server.PrivateKeyID, service.TeamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get private key: %w", err)
	}
	if privateKey == nil {
		return nil, fmt.Errorf("private key not found")
	}

	// Parse the Docker Compose configuration
	namingGenerator := generator.NewNamingGenerator(service.ID, service.TeamID, service.ServerID, service.ProjectID)
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
	sshClient, err := h.ConnectionPool.GetSSHConnection(connectionID, server.Host, server.Port, server.User, []byte(privateKey.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to get Docker connection: %w", err)
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

	// Execute the rebuild operation asynchronously in a goroutine
	// Use context.Background() to ensure operation continues even if client disconnects
	go func() {
		if err := dockerHandler.RebuildDockerCompose(context.Background()); err != nil {
			zap.L().Error("Failed to rebuild Docker compose", zap.Error(err))
		}
	}()

	// Return the streaming result immediately
	return &streamChan, nil
}

// UpdateServiceComposeFromRequest updates a compose config model with new values from update request
func updateServiceComposeFromRequest(existingConfig models.ServiceComposeConfig, updateServiceComposeRequest updateServiceComposeRequest) models.ServiceComposeConfig {
	if updateServiceComposeRequest.ComposeFile != nil {
		existingConfig.ComposeFile = *updateServiceComposeRequest.ComposeFile
	}
	existingConfig.UpdatedAt = time.Now()

	return existingConfig
}
