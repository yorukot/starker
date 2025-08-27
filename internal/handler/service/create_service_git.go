package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/handler/service/utils/git"
	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/encrypt"
	"github.com/yorukot/starker/pkg/generator"
	"github.com/yorukot/starker/pkg/response"
)

// +----------------------------------------------+
// | Create Service from Git Repository           |
// +----------------------------------------------+

// CreateServiceGit godoc
// @Summary Create a service from a Git repository
// @Description Creates a new service by cloning a Git repository and extracting Docker Compose configuration
// @Tags service
// @Accept json
// @Produce text/plain
// @Param teamID path string true "Team ID"
// @Param projectID path string true "Project ID"
// @Param request body CreateServiceGitRequest true "Git service creation request"
// @Success 200 {string} string "Server-Sent Events stream with git workflow progress"
// @Failure 400 {object} response.ErrorResponse "Invalid request body, team access denied, or project not found"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/projects/{projectID}/services/git [post]
// @Security BearerAuth
func (h *ServiceHandler) CreateServiceGit(w http.ResponseWriter, r *http.Request) {
	// Get teamID and projectID from the request
	teamID := chi.URLParam(r, "teamID")
	projectID := chi.URLParam(r, "projectID")

	// Get the service request from the request body
	var createGitRequest CreateServiceGitRequest
	if err := json.NewDecoder(r.Body).Decode(&createGitRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	// Validate the request body
	if err := serviceGitValidate(createGitRequest); err != nil {
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

	// Check if the project exists
	project, err := repository.GetProject(r.Context(), tx, teamID, projectID)
	if err != nil {
		zap.L().Error("Failed to get project", zap.Error(err))
		response.RespondWithError(w, http.StatusBadRequest, "Project not found", "PROJECT_NOT_FOUND")
		return
	}

	// Check if the server exists and get its details
	server, err := repository.GetServerByID(r.Context(), tx, createGitRequest.ServerID, teamID)
	if err != nil {
		zap.L().Error("Failed to get server", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get server", "FAILED_TO_GET_SERVER")
		return
	}

	if server == nil {
		response.RespondWithError(w, http.StatusBadRequest, "Server not found", "SERVER_NOT_FOUND")
		return
	}

	// Get the private key for SSH connection
	privateKey, err := repository.GetPrivateKeyByID(r.Context(), tx, server.PrivateKeyID, teamID)
	if err != nil {
		zap.L().Error("Failed to get private key", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get private key", "FAILED_TO_GET_PRIVATE_KEY")
		return
	}

	// All validation passed, now start the SSE stream
	// Set headers for Server-Sent Events
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Send initial response
	fmt.Fprint(w, "data: {\"type\":\"log\",\"message\":\"Starting git service creation workflow...\"}\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	fmt.Fprint(w, "data: {\"type\":\"log\",\"message\":\"Validation completed, starting git workflow...\"}\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Generate service and git source models
	service := generateGitService(createGitRequest, teamID, createGitRequest.ServerID, project.ID)
	gitSource := generateServiceSourceGit(service.ID, createGitRequest)

	// Create the service and git source records first
	if err = repository.CreateService(r.Context(), tx, service); err != nil {
		zap.L().Error("Failed to create service", zap.Error(err))
		fmt.Fprint(w, "data: {\"type\":\"error\",\"message\":\"Failed to create service\"}\n\n")
		return
	}

	if err = repository.CreateServiceSourceGit(r.Context(), tx, gitSource); err != nil {
		zap.L().Error("Failed to create git source", zap.Error(err))
		fmt.Fprint(w, "data: {\"type\":\"error\",\"message\":\"Failed to create git source\"}\n\n")
		return
	}

	// Commit the transaction early so service exists in DB
	repository.CommitTransaction(tx, r.Context())

	// Now execute the git workflow
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	// Generate connection ID for this operation
	namingGen := generator.NewNamingGenerator(teamID, projectID, service.ID)
	connectionID := namingGen.ConnectionID()
	host := fmt.Sprintf("%s@%s:%s", server.User, server.IP, server.Port)

	// Build git workflow config
	workflowConfig := git.BuildGitWorkflowConfig(
		service.ID,
		&gitSource,
		h.DockerPool,
		connectionID,
		host,
		[]byte(privateKey.PrivateKey),
	)

	// Execute the git workflow
	workflowResult, err := git.ExecuteGitWorkflow(ctx, workflowConfig)
	if err != nil {
		zap.L().Error("Failed to start git workflow", zap.Error(err))
		fmt.Fprintf(w, "data: {\"type\":\"error\",\"message\":\"Failed to start git workflow: %s\"}\n\n", err.Error())
		return
	}

	// Stream the workflow progress
	success := h.streamGitWorkflowProgress(w, workflowResult, service.ID)

	if success {
		fmt.Fprintf(w, "data: {\"type\":\"success\",\"message\":\"Git service created successfully\",\"service_id\":\"%s\"}\n\n", service.ID)
	}
}

// streamGitWorkflowProgress streams the git workflow progress via SSE
func (h *ServiceHandler) streamGitWorkflowProgress(w http.ResponseWriter, workflowResult *git.GitWorkflowResult, serviceID string) bool {
	for {
		select {
		case log, ok := <-workflowResult.LogChan:
			if !ok {
				continue
			}
			fmt.Fprintf(w, "data: {\"type\":\"log\",\"message\":\"%s\"}\n\n", log)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

		case err, ok := <-workflowResult.ErrorChan:
			if !ok {
				continue
			}
			zap.L().Error("Git workflow error", zap.Error(err))
			fmt.Fprintf(w, "data: {\"type\":\"error\",\"message\":\"%s\"}\n\n", err.Error())
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			return false

		case <-workflowResult.DoneChan:
			if workflowResult.GetFinalError() != nil {
				fmt.Fprintf(w, "data: {\"type\":\"error\",\"message\":\"Git workflow failed: %s\"}\n\n", workflowResult.GetFinalError().Error())
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
				return false
			}

			// Workflow completed successfully, save the compose file
			if err := h.saveComposeFileFromGitWorkflow(serviceID, workflowResult.ComposeFile); err != nil {
				zap.L().Error("Failed to save compose file", zap.Error(err))
				fmt.Fprintf(w, "data: {\"type\":\"error\",\"message\":\"Failed to save compose file: %s\"}\n\n", err.Error())
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
				return false
			}

			fmt.Fprint(w, "data: {\"type\":\"log\",\"message\":\"Docker Compose file saved successfully\"}\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			return true
		}
	}
}

// saveComposeFileFromGitWorkflow saves the extracted compose file to the database
func (h *ServiceHandler) saveComposeFileFromGitWorkflow(serviceID, composeFile string) error {
	ctx := context.Background()

	// Start a new transaction for saving the compose file
	tx, err := repository.StartTransaction(h.DB, ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.DeferRollback(tx, ctx)

	// Generate compose config model
	composeConfig := generateServiceComposeConfig(serviceID, composeFile)

	// Create the compose config
	if err = repository.CreateServiceComposeConfig(ctx, tx, composeConfig); err != nil {
		return fmt.Errorf("failed to create compose config: %w", err)
	}

	// Commit the transaction
	repository.CommitTransaction(tx, ctx)
	return nil
}

// CreateServiceGitRequest represents the request body for creating a service from git repository
type CreateServiceGitRequest struct {
	Name                  string  `json:"name" validate:"required,min=3,max=255" example:"my-app"`                                      // Service name
	Description           *string `json:"description,omitempty" validate:"omitempty,max=500" example:"Application from Git"`            // Optional service description
	ServerID              string  `json:"server_id" validate:"required" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`                           // Server ID where service will be deployed
	RepoURL               string  `json:"repo_url" validate:"required,url" example:"https://github.com/user/repo.git"`                  // Git repository URL
	Branch                string  `json:"branch" validate:"required" example:"main"`                                                    // Git branch to deploy
	DockerComposeFilePath *string `json:"docker_compose_file_path,omitempty" validate:"omitempty,max=255" example:"docker-compose.yml"` // Path to docker-compose file in repo
	AutoDeploy            bool    `json:"auto_deploy" example:"true"`                                                                   // Enable auto-deployment on Git changes
}

// serviceGitValidate validates the create service git request
func serviceGitValidate(request CreateServiceGitRequest) error {
	return validator.New().Struct(request)
}

// generateGitService generates a service model for git-based service creation
func generateGitService(request CreateServiceGitRequest, teamID, serverID, projectID string) models.Service {
	now := time.Now()

	return models.Service{
		ID:          ksuid.New().String(),
		TeamID:      teamID,
		ServerID:    serverID,
		ProjectID:   projectID,
		Name:        request.Name,
		Description: request.Description,
		Type:        "git", // New service type for git-based services
		State:       models.ServiceStateStopped,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// generateServiceSourceGit generates a git source model
func generateServiceSourceGit(serviceID string, request CreateServiceGitRequest) models.ServiceSourceGit {
	now := time.Now()

	return models.ServiceSourceGit{
		ID:                    ksuid.New().String(),
		ServiceID:             serviceID,
		RepoURL:               request.RepoURL,
		Branch:                request.Branch,
		AutoDeploy:            request.AutoDeploy,
		DockerComposeFilePath: request.DockerComposeFilePath,
		WebhookSecret:         generateWebhookSecret(), // Generate a random webhook secret
		CreatedAt:             &now,
		UpdatedAt:             &now,
	}
}

// generateWebhookSecret generates a random webhook secret
func generateWebhookSecret() string {
	secret, err := encrypt.GenerateRandomString(32)
	if err != nil {
		// Fallback to KSUID if random generation fails
		return ksuid.New().String()
	}
	return secret
}
