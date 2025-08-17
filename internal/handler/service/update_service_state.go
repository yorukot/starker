// +----------------------------------------------+
// | Update Service Status                        |
// +----------------------------------------------+

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/handler/service/utils"
	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/response"
	"github.com/yorukot/starker/pkg/sshpool"
)

type updateServiceStateRequest struct {
	State string `json:"state" validate:"required,oneof=start stop restart"`
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

	// Use service layer to handle the complete operation
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

	// Execute the service operation
	result, err := h.executeServiceOperation(r.Context(), newState, serviceID, teamID, projectID)
	if err != nil {
		zap.L().Error("Failed to execute service command", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to execute service command", "FAILED_TO_EXECUTE_COMMAND")
		return
	}

	// Update service to initial status before streaming
	service.Status = result.InitialStatus
	if err := repository.UpdateService(r.Context(), *h.Tx, *service); err != nil {
		zap.L().Error("Failed to update initial service status", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to update service status", "FAILED_TO_UPDATE_SERVICE_STATUS")
		return
	}

	h.streamServiceOutputWithUpdate(w, result, service, r.Context())

	// Commit transaction after successful completion
	repository.CommitTransaction(tx, r.Context())
}

// serviceOperationResult contains the result of a service operation
type serviceOperationResult struct {
	StreamResult  *sshpool.StreamingCommandResult
	InitialStatus models.ServiceStatus
	SuccessStatus models.ServiceStatus
	FailureStatus models.ServiceStatus
}

// executeServiceOperation executes a service operation (start/stop/restart)
func (h *ServiceHandler) executeServiceOperation(ctx context.Context, operation, serviceID, teamID, projectID string) (*serviceOperationResult, error) {
	var streamResult *sshpool.StreamingCommandResult
	var initialStatus, successStatus, failureStatus models.ServiceStatus
	var err error

	switch operation {
	case "start":
		initialStatus = models.ServiceStatusStarting
		successStatus = models.ServiceStatusRunning
		failureStatus = models.ServiceStatusStopped
		streamResult, err = utils.StartService(ctx, serviceID, teamID, projectID, *h.Tx, h.SSHPool)
	case "stop":
		initialStatus = models.ServiceStatusStopping
		successStatus = models.ServiceStatusStopped
		failureStatus = models.ServiceStatusRunning
		streamResult, err = utils.StopService(ctx, serviceID, teamID, projectID, *h.Tx, h.SSHPool)
	case "restart":
		initialStatus = models.ServiceStatusRestarting
		successStatus = models.ServiceStatusRunning
		failureStatus = models.ServiceStatusStopped
		streamResult, err = utils.RestartService(ctx, serviceID, teamID, projectID, *h.Tx, h.SSHPool)
	default:
		return nil, fmt.Errorf("invalid operation: %s", operation)
	}

	if err != nil {
		return nil, err
	}

	return &serviceOperationResult{
		StreamResult:  streamResult,
		InitialStatus: initialStatus,
		SuccessStatus: successStatus,
		FailureStatus: failureStatus,
	}, nil
}

// escapeJSONString escapes a string for safe inclusion in JSON
func escapeJSONString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

// StreamServiceOutputWithUpdate streams service operation output via SSE and updates final status
func (h *ServiceHandler) streamServiceOutputWithUpdate(w http.ResponseWriter, result *serviceOperationResult, service *models.Service, ctx context.Context) {
	// Set up SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Stream the command output via SSE in real-time
	for {
		select {
		case line, ok := <-result.StreamResult.StdoutChan:
			if !ok {
				continue
			}
			fmt.Fprintf(w, "data: {\"type\": \"stdout\", \"message\": \"%s\"}\n\n", escapeJSONString(line))
			w.(http.Flusher).Flush()
		case line, ok := <-result.StreamResult.StderrChan:
			if !ok {
				continue
			}
			fmt.Fprintf(w, "data: {\"type\": \"stderr\", \"message\": \"%s\"}\n\n", escapeJSONString(line))
			w.(http.Flusher).Flush()
		case err, ok := <-result.StreamResult.ErrorChan:
			if !ok {
				continue
			}
			fmt.Fprintf(w, "data: {\"type\": \"error\", \"message\": \"%s\"}\n\n", escapeJSONString(err.Error()))
			w.(http.Flusher).Flush()
		case <-result.StreamResult.DoneChan:
			// Command finished, determine final status
			finalError := result.StreamResult.GetFinalError()
			if finalError != nil {
				fmt.Fprintf(w, "data: {\"type\": \"error\", \"message\": \"%s\"}\n\n", escapeJSONString(finalError.Error()))
				service.Status = result.FailureStatus
			} else {
				service.Status = result.SuccessStatus
			}

			// Update final service status
			if err := repository.UpdateService(ctx, *h.Tx, *service); err != nil {
				zap.L().Error("Failed to update final service status", zap.Error(err))
				fmt.Fprintf(w, "data: {\"type\": \"error\", \"message\": \"Failed to update service status\"}\n\n")
				w.(http.Flusher).Flush()
				return
			}

			fmt.Fprintf(w, "data: {\"type\": \"status\", \"status\": \"completed\", \"final_state\": \"%s\"}\n\n", service.Status)
			w.(http.Flusher).Flush()
			return
		}
	}
}
