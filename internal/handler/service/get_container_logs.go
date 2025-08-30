// +----------------------------------------------+
// | Get Container Logs                           |
// +----------------------------------------------+

package service

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/core"
	"github.com/yorukot/starker/internal/core/dockerutils"
	"github.com/yorukot/starker/internal/handler/service/utils"
	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/generator"
	"github.com/yorukot/starker/pkg/response"
)

// GetContainerLogs godoc
// @Summary Get container logs with SSE streaming
// @Description Retrieves Docker container logs for a specific service container with real-time streaming via Server-Sent Events
// @Tags service
// @Accept json
// @Produce text/event-stream
// @Param teamID path string true "Team ID"
// @Param projectID path string true "Project ID"
// @Param serviceID path string true "Service ID"
// @Param containerID path string true "Container ID (from service_containers table)"
// @Param follow query bool false "Follow log output (stream continuously)" default(false)
// @Param tail query string false "Number of lines to show from end of logs" default("100")
// @Param timestamps query bool false "Include timestamps in log output" default(false)
// @Param since query string false "Show logs since timestamp (RFC3339 format)" example("2023-01-01T00:00:00Z")
// @Success 200 {string} string "SSE stream of container logs"
// @Failure 400 {object} response.ErrorResponse "Team access denied, service/container not found, or invalid parameters"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/projects/{projectID}/services/{serviceID}/containers/{contUNAUTHORIZEDainerID}/logs [get]
// @Security BearerAuth
func (h *ServiceHandler) GetContainerLogs(w http.ResponseWriter, r *http.Request) {
	// Get URL parameters
	teamID := chi.URLParam(r, "teamID")
	projectID := chi.URLParam(r, "projectID")
	serviceID := chi.URLParam(r, "serviceID")
	containerID := chi.URLParam(r, "containerID")

	// Get userID
	userID := r.Context().Value(middleware.UserIDKey).(string)

	// Parse query parameters for log options
	logOptions := dockerutils.LogOptions{
		Follow:     r.URL.Query().Get("follow") != "false", // Default to true for real-time streaming
		Tail:       r.URL.Query().Get("tail"),
		Timestamps: r.URL.Query().Get("timestamps") == "true",
	}

	// Set default tail if not specified
	if logOptions.Tail == "" {
		logOptions.Tail = "100"
	}

	// Parse since timestamp if provided
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		since, err := time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			response.RespondWithError(w, http.StatusBadRequest, "Invalid since timestamp format", "INVALID_SINCE_TIMESTAMP")
			return
		}
		logOptions.Since = since
	}

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

	// Verify service exists and belongs to the team/project
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

	// Get the specific container by ID
	container, err := repository.GetServiceContainerByID(r.Context(), tx, containerID, serviceID)
	if err != nil {
		zap.L().Error("Failed to find container", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to find container", "FAILED_TO_FIND_CONTAINER")
		return
	}
	if container == nil {
		response.RespondWithError(w, http.StatusBadRequest, "Container not found", "CONTAINER_NOT_FOUND")
		return
	}

	// Check if container has a Docker container ID
	if container.ContainerID == nil || *container.ContainerID == "" {
		response.RespondWithError(w, http.StatusBadRequest, "Container has no Docker container ID", "NO_DOCKER_CONTAINER_ID")
		return
	}

	// Get server details for connection
	server, err := repository.GetServerByID(r.Context(), tx, service.ServerID, service.TeamID)
	if err != nil {
		zap.L().Error("Failed to get server", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get server", "FAILED_TO_GET_SERVER")
		return
	}
	if server == nil {
		response.RespondWithError(w, http.StatusBadRequest, "Server not found", "SERVER_NOT_FOUND")
		return
	}

	// Get private key for SSH connection
	privateKey, err := repository.GetPrivateKeyByID(r.Context(), tx, server.PrivateKeyID, service.TeamID)
	if err != nil {
		zap.L().Error("Failed to get private key", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get private key", "FAILED_TO_GET_PRIVATE_KEY")
		return
	}
	if privateKey == nil {
		response.RespondWithError(w, http.StatusBadRequest, "Private key not found", "PRIVATE_KEY_NOT_FOUND")
		return
	}

	// Get Docker client from connection pool
	namingGenerator := generator.NewNamingGenerator(service.ID, service.TeamID, service.ServerID)
	connectionID := namingGenerator.ConnectionID()
	sshHost := fmt.Sprintf("%s@%s:%s", server.User, server.IP, server.Port)
	dockerClient, err := h.ConnectionPool.GetDockerConnection(connectionID, sshHost, []byte(privateKey.PrivateKey))
	if err != nil {
		zap.L().Error("Failed to get Docker connection", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get Docker connection", "FAILED_TO_GET_DOCKER_CONNECTION")
		return
	}

	// Create Docker handler with proper StreamChan initialization for log operations
	dockerHandler := &dockerutils.DockerHandler{
		Client:     dockerClient,
		StreamChan: core.NewStreamChan(),
	}

	// Commit transaction since we're moving to streaming
	repository.CommitTransaction(tx, r.Context())

	// Get container logs
	logsReader, err := dockerHandler.GetContainerLogs(r.Context(), *container.ContainerID, logOptions)
	if err != nil {
		zap.L().Error("Failed to get container logs", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get container logs", "FAILED_TO_GET_CONTAINER_LOGS")
		return
	}

	defer logsReader.Close()
	// Stream container logs using utility function
	utils.StreamContainerLogs(r.Context(), w, logsReader, container.ContainerName)
}
