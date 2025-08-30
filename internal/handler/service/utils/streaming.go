package utils

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/core"
	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/response"
)

// StreamServiceOutputWithUpdate handles SSE streaming of Docker operation progress
func StreamServiceOutputWithUpdate(ctx context.Context, w http.ResponseWriter, streamChan *core.StreamChan, service *models.Service, tx *pgx.Tx, operation string) (done bool) {
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

		case logMsg := <-streamChan.LogChan:
			// Stream log message
			data, _ := json.Marshal(map[string]interface{}{
				"message": logMsg.Message,
				"type":    string(logMsg.Type),
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

		case errMsg := <-streamChan.ErrChan:
			// Stream error message
			data, _ := json.Marshal(map[string]interface{}{
				"message": errMsg.Message,
				"type":    string(errMsg.Type),
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

		case progressMsg := <-streamChan.ProgressChan:
			// Stream progress message
			data, _ := json.Marshal(map[string]interface{}{
				"message": progressMsg.Message,
				"type":    string(progressMsg.Type),
				"data":    progressMsg.Data,
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

		case finalErr := <-streamChan.FinalError:
			// Operation failed - rollback transaction
			zap.L().Error("Docker operation failed", zap.Error(finalErr))

			// Update service state to stopped (rollback)
			service.State = models.ServiceStateStopped
			if updateErr := repository.UpdateService(ctx, *tx, *service); updateErr != nil {
				zap.L().Error("Failed to rollback service state", zap.Error(updateErr))
			}

			data, _ := json.Marshal(map[string]interface{}{
				"message": fmt.Sprintf("Operation failed: %v", finalErr),
				"type":    "error",
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			return

		case <-streamChan.DoneChan:
			// Operation completed successfully
			zap.L().Info("Docker operation completed successfully")

			// Update service state based on operation
			var successMessage string
			var finalState string

			switch operation {
			case "start":
				service.State = models.ServiceStateRunning
				service.LastDeployedAt = &[]time.Time{time.Now()}[0]
				successMessage = "Service started successfully"
				finalState = "running"
			case "stop":
				service.State = models.ServiceStateStopped
				successMessage = "Service stopped successfully"
				finalState = "stopped"
			case "restart":
				service.State = models.ServiceStateRunning
				service.LastDeployedAt = &[]time.Time{time.Now()}[0]
				successMessage = "Service restarted successfully"
				finalState = "running"
			default:
				service.State = models.ServiceStateRunning
				successMessage = "Service operation completed successfully"
				finalState = "running"
			}

			if err := repository.UpdateService(ctx, *tx, *service); err != nil {
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
			if err := (*tx).Commit(ctx); err != nil {
				zap.L().Error("Failed to commit transaction", zap.Error(err))
				data, _ := json.Marshal(map[string]interface{}{
					"message": "Failed to commit database transaction",
					"type":    "error",
				})
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
				return
			}

			// Send success completion event with operation-specific message
			data, _ := json.Marshal(map[string]interface{}{
				"message": successMessage,
				"type":    core.LogTypeInfo,
				"state":   finalState,
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			return
		}
	}
}

// StreamContainerLogs handles SSE streaming of Docker container logs
func StreamContainerLogs(ctx context.Context, w http.ResponseWriter, logsReader io.ReadCloser, containerName string) {
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

	// Helper function to send SSE events
	sendEvent := func(logMsg core.LogMessage) {
		data, _ := json.Marshal(logMsg)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	// Send initial event
	sendEvent(core.LogInfo(fmt.Sprintf("Starting log stream for container: %s", containerName)))

	// Stream logs line by line
	scanner := bufio.NewScanner(logsReader)
	lineNumber := 0
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			// Client disconnected
			zap.L().Info("Client disconnected from log stream")
			return
		default:
			line := scanner.Text()
			lineNumber++

			// Send log line as SSE event with container metadata
			logData := map[string]any{
				"line":        line,
				"line_number": lineNumber,
				"container":   containerName,
			}
			logMsg := core.LogMessage{
				Type:    core.LogTypeInfo,
				Message: line,
				Data:    logData,
			}
			sendEvent(logMsg)
		}
	}

	// Check for scanning errors
	if err := scanner.Err(); err != nil && err != io.EOF {
		zap.L().Error("Error reading container logs", zap.Error(err))
		sendEvent(core.LogError(fmt.Sprintf("Error reading logs: %v", err)))
		return
	}

	// Send completion event
	completionData := map[string]any{
		"line_count": lineNumber,
		"completed":  true,
		"container":  containerName,
	}
	completionMsg := core.LogMessage{
		Type:    core.LogTypeInfo,
		Message: fmt.Sprintf("Log stream completed. Total lines: %d", lineNumber),
		Data:    completionData,
	}
	sendEvent(completionMsg)
}
