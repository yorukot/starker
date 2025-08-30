package utils

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

// StreamContainerLogs handles real-time SSE streaming of Docker container logs
func StreamContainerLogs(ctx context.Context, w http.ResponseWriter, logsReader io.ReadCloser, containerName string) {
	flusher := setupSSEHeaders(w)
	if flusher == nil {
		return
	}

	sendEvent := createEventSender(flusher, w)
	sendEvent(core.LogInfo(fmt.Sprintf("Starting log stream for container: %s", containerName)))

	lineNumber := 0
	header := make([]byte, 8)

	for {
		if ctx.Err() != nil {
			zap.L().Info("Client disconnected from log stream")
			return
		}

		// Read Docker header
		n, err := io.ReadFull(logsReader, header)
		if err != nil {
			handleReadError(err, n, lineNumber, containerName, sendEvent)
			return
		}

		// Parse header and read payload
		streamType := header[0]
		payloadSize := binary.BigEndian.Uint32(header[4:8])

		if payloadSize == 0 {
			continue
		}

		payload, err := readPayload(logsReader, payloadSize)
		if err != nil {
			zap.L().Error("Failed to read Docker log payload", zap.Error(err))
			sendEvent(core.LogError(fmt.Sprintf("Error reading log payload: %v", err)))
			return
		}

		lineNumber = processLogPayload(payload, streamType, containerName, lineNumber, sendEvent)
	}
}

// setupSSEHeaders configures SSE headers and returns flusher
func setupSSEHeaders(w http.ResponseWriter) http.Flusher {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		zap.L().Error("Streaming unsupported")
		response.RespondWithError(w, http.StatusInternalServerError, "Streaming unsupported", "STREAMING_UNSUPPORTED")
		return nil
	}
	return flusher
}

// createEventSender returns a function to send SSE events
func createEventSender(flusher http.Flusher, w http.ResponseWriter) func(core.LogMessage) {
	return func(logMsg core.LogMessage) {
		data, _ := json.Marshal(logMsg)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

// handleReadError handles errors when reading Docker log headers
func handleReadError(err error, n int, lineNumber int, containerName string, sendEvent func(core.LogMessage)) {
	if err == io.EOF || (err == io.ErrUnexpectedEOF && n == 0) {
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
		return
	}

	zap.L().Error("Failed to read Docker log header", zap.Error(err))
	sendEvent(core.LogError(fmt.Sprintf("Error reading log header: %v", err)))
}

// readPayload reads the Docker log payload of specified size
func readPayload(reader io.Reader, size uint32) ([]byte, error) {
	payload := make([]byte, size)
	_, err := io.ReadFull(reader, payload)
	return payload, err
}

// processLogPayload processes payload and sends individual log lines
func processLogPayload(payload []byte, streamType byte, containerName string, startLineNumber int, sendEvent func(core.LogMessage)) int {
	logText := string(payload)
	lines := strings.Split(logText, "\n")
	lineNumber := startLineNumber

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		lineNumber++
		streamName, logType := getStreamInfo(streamType)

		logData := map[string]any{
			"line":        line,
			"line_number": lineNumber,
			"container":   containerName,
			"stream":      streamName,
		}
		logMsg := core.LogMessage{
			Type:    logType,
			Message: line,
			Data:    logData,
		}
		sendEvent(logMsg)
	}

	return lineNumber
}

// getStreamInfo returns stream name and log type based on Docker stream type
func getStreamInfo(streamType byte) (string, core.LogType) {
	switch streamType {
	case 1: // stdout
		return "stdout", core.LogTypeInfo
	case 2: // stderr
		return "stderr", core.LogTypeError
	default: // unknown stream type, treat as stdout
		return "stdout", core.LogTypeInfo
	}
}
