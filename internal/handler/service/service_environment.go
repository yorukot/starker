package service

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/response"
)

// +----------------------------------------------+
// | Get Service Environments                     |
// +----------------------------------------------+

// GetServiceEnvironments godoc
// @Summary Get all environment variables for a service
// @Description Retrieves all environment variables for a specific service within a team and project
// @Tags service
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param projectID path string true "Project ID"
// @Param serviceID path string true "Service ID"
// @Success 200 {object} response.SuccessResponse{data=[]models.ServiceEnvironment} "Environment variables retrieved successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request or team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 404 {object} response.ErrorResponse "Service not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/projects/{projectID}/services/{serviceID}/env [get]
// @Security BearerAuth
func (h *ServiceHandler) GetServiceEnvironments(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamID")
	projectID := chi.URLParam(r, "projectID")
	serviceID := chi.URLParam(r, "serviceID")

	userID := r.Context().Value(middleware.UserIDKey).(string)

	tx, err := repository.StartTransaction(h.DB, r.Context())
	if err != nil {
		zap.L().Error("Failed to begin transaction", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
		return
	}
	defer repository.DeferRollback(tx, r.Context())

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

	service, err := repository.GetServiceByID(r.Context(), tx, serviceID, teamID, projectID)
	if err != nil {
		zap.L().Error("Failed to find service", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to find service", "FAILED_TO_FIND_SERVICE")
		return
	}
	if service == nil {
		response.RespondWithError(w, http.StatusNotFound, "Service not found", "SERVICE_NOT_FOUND")
		return
	}

	environments, err := repository.GetServiceEnvironments(r.Context(), tx, serviceID)
	if err != nil {
		zap.L().Error("Failed to get service environments", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get service environments", "FAILED_TO_GET_SERVICE_ENVIRONMENTS")
		return
	}

	repository.CommitTransaction(tx, r.Context())
	response.RespondWithJSON(w, http.StatusOK, environments)
}

// +----------------------------------------------+
// | Create Service Environment                   |
// +----------------------------------------------+

type createServiceEnvironmentRequest struct {
	Key   string `json:"key" validate:"required,min=1,max=255"`
	Value string `json:"value" validate:"required"`
}

// CreateServiceEnvironment godoc
// @Summary Create a new environment variable for a service
// @Description Creates a new environment variable for a specific service within a team and project
// @Tags service
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param projectID path string true "Project ID"
// @Param serviceID path string true "Service ID"
// @Param request body createServiceEnvironmentRequest true "Environment variable creation request"
// @Success 201 {object} response.SuccessResponse{data=models.ServiceEnvironment} "Environment variable created successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body or team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 404 {object} response.ErrorResponse "Service not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/projects/{projectID}/services/{serviceID}/env [post]
// @Security BearerAuth
func (h *ServiceHandler) CreateServiceEnvironment(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamID")
	projectID := chi.URLParam(r, "projectID")
	serviceID := chi.URLParam(r, "serviceID")

	var createRequest createServiceEnvironmentRequest
	if err := json.NewDecoder(r.Body).Decode(&createRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	if err := validator.New().Struct(createRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	userID := r.Context().Value(middleware.UserIDKey).(string)

	tx, err := repository.StartTransaction(h.DB, r.Context())
	if err != nil {
		zap.L().Error("Failed to begin transaction", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
		return
	}
	defer repository.DeferRollback(tx, r.Context())

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

	service, err := repository.GetServiceByID(r.Context(), tx, serviceID, teamID, projectID)
	if err != nil {
		zap.L().Error("Failed to find service", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to find service", "FAILED_TO_FIND_SERVICE")
		return
	}
	if service == nil {
		response.RespondWithError(w, http.StatusNotFound, "Service not found", "SERVICE_NOT_FOUND")
		return
	}

	now := time.Now()
	env := models.ServiceEnvironment{
		ServiceID: serviceID,
		Key:       createRequest.Key,
		Value:     createRequest.Value,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := repository.CreateServiceEnvironment(r.Context(), tx, env); err != nil {
		zap.L().Error("Failed to create service environment", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to create service environment", "FAILED_TO_CREATE_SERVICE_ENVIRONMENT")
		return
	}

	repository.CommitTransaction(tx, r.Context())
	response.RespondWithJSON(w, http.StatusCreated, env)
}

// +----------------------------------------------+
// | Create Service Environments (Batch)         |
// +----------------------------------------------+

type createServiceEnvironmentsRequest struct {
	Environments []createServiceEnvironmentRequest `json:"environments" validate:"required,min=1,dive"`
}

// CreateServiceEnvironments godoc
// @Summary Create multiple environment variables for a service
// @Description Creates multiple environment variables for a specific service within a team and project
// @Tags service
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param projectID path string true "Project ID"
// @Param serviceID path string true "Service ID"
// @Param request body createServiceEnvironmentsRequest true "Batch environment variables creation request"
// @Success 201 {object} response.SuccessResponse{data=[]models.ServiceEnvironment} "Environment variables created successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body or team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 404 {object} response.ErrorResponse "Service not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/projects/{projectID}/services/{serviceID}/env/batch [post]
// @Security BearerAuth
func (h *ServiceHandler) CreateServiceEnvironments(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamID")
	projectID := chi.URLParam(r, "projectID")
	serviceID := chi.URLParam(r, "serviceID")

	var createRequest createServiceEnvironmentsRequest
	if err := json.NewDecoder(r.Body).Decode(&createRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	if err := validator.New().Struct(createRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	userID := r.Context().Value(middleware.UserIDKey).(string)

	tx, err := repository.StartTransaction(h.DB, r.Context())
	if err != nil {
		zap.L().Error("Failed to begin transaction", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
		return
	}
	defer repository.DeferRollback(tx, r.Context())

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

	service, err := repository.GetServiceByID(r.Context(), tx, serviceID, teamID, projectID)
	if err != nil {
		zap.L().Error("Failed to find service", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to find service", "FAILED_TO_FIND_SERVICE")
		return
	}
	if service == nil {
		response.RespondWithError(w, http.StatusNotFound, "Service not found", "SERVICE_NOT_FOUND")
		return
	}

	now := time.Now()
	var environments []models.ServiceEnvironment
	for _, envReq := range createRequest.Environments {
		env := models.ServiceEnvironment{
			ServiceID: serviceID,
			Key:       envReq.Key,
			Value:     envReq.Value,
			CreatedAt: now,
			UpdatedAt: now,
		}
		environments = append(environments, env)
	}

	if err := repository.CreateServiceEnvironments(r.Context(), tx, environments); err != nil {
		zap.L().Error("Failed to create service environments", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to create service environments", "FAILED_TO_CREATE_SERVICE_ENVIRONMENTS")
		return
	}

	repository.CommitTransaction(tx, r.Context())
	response.RespondWithJSON(w, http.StatusCreated, environments)
}

// +----------------------------------------------+
// | Update Service Environment                   |
// +----------------------------------------------+

type updateServiceEnvironmentRequest struct {
	Key   *string `json:"key,omitempty" validate:"omitempty,min=1,max=255"`
	Value *string `json:"value,omitempty" validate:"omitempty"`
}

// UpdateServiceEnvironment godoc
// @Summary Update an environment variable for a service
// @Description Updates an existing environment variable for a specific service within a team and project
// @Tags service
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param projectID path string true "Project ID"
// @Param serviceID path string true "Service ID"
// @Param envID path int true "Environment Variable ID"
// @Param request body updateServiceEnvironmentRequest true "Environment variable update request"
// @Success 200 {object} response.SuccessResponse{data=models.ServiceEnvironment} "Environment variable updated successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body or team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 404 {object} response.ErrorResponse "Service or environment variable not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/projects/{projectID}/services/{serviceID}/env/{envID} [patch]
// @Security BearerAuth
func (h *ServiceHandler) UpdateServiceEnvironment(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamID")
	projectID := chi.URLParam(r, "projectID")
	serviceID := chi.URLParam(r, "serviceID")
	envIDStr := chi.URLParam(r, "envID")

	envID, err := strconv.ParseInt(envIDStr, 10, 64)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid environment variable ID", "INVALID_ENV_ID")
		return
	}

	var updateRequest updateServiceEnvironmentRequest
	if err := json.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	if err := validator.New().Struct(updateRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	userID := r.Context().Value(middleware.UserIDKey).(string)

	tx, err := repository.StartTransaction(h.DB, r.Context())
	if err != nil {
		zap.L().Error("Failed to begin transaction", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
		return
	}
	defer repository.DeferRollback(tx, r.Context())

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

	service, err := repository.GetServiceByID(r.Context(), tx, serviceID, teamID, projectID)
	if err != nil {
		zap.L().Error("Failed to find service", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to find service", "FAILED_TO_FIND_SERVICE")
		return
	}
	if service == nil {
		response.RespondWithError(w, http.StatusNotFound, "Service not found", "SERVICE_NOT_FOUND")
		return
	}

	env, err := repository.GetServiceEnvironment(r.Context(), tx, envID, serviceID)
	if err != nil {
		zap.L().Error("Failed to find environment variable", zap.Error(err))
		response.RespondWithError(w, http.StatusNotFound, "Environment variable not found", "ENVIRONMENT_VARIABLE_NOT_FOUND")
		return
	}

	updatedEnv := updateEnvironmentFromRequest(*env, updateRequest)

	if err := repository.UpdateServiceEnvironment(r.Context(), tx, updatedEnv); err != nil {
		zap.L().Error("Failed to update service environment", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to update service environment", "FAILED_TO_UPDATE_SERVICE_ENVIRONMENT")
		return
	}

	repository.CommitTransaction(tx, r.Context())
	response.RespondWithJSON(w, http.StatusOK, updatedEnv)
}

// +----------------------------------------------+
// | Update Service Environments (Batch)         |
// +----------------------------------------------+

type updateServiceEnvironmentItem struct {
	ID    int64   `json:"id" validate:"required"`
	Key   *string `json:"key,omitempty" validate:"omitempty,min=1,max=255"`
	Value *string `json:"value,omitempty" validate:"omitempty"`
}

type updateServiceEnvironmentsRequest struct {
	Environments []updateServiceEnvironmentItem `json:"environments" validate:"required,min=1,dive"`
}

// UpdateServiceEnvironments godoc
// @Summary Update multiple environment variables for a service
// @Description Updates multiple existing environment variables for a specific service within a team and project
// @Tags service
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param projectID path string true "Project ID"
// @Param serviceID path string true "Service ID"
// @Param request body updateServiceEnvironmentsRequest true "Batch environment variables update request"
// @Success 200 {object} response.SuccessResponse{data=[]models.ServiceEnvironment} "Environment variables updated successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body or team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 404 {object} response.ErrorResponse "Service or environment variable not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/projects/{projectID}/services/{serviceID}/env/batch [patch]
// @Security BearerAuth
func (h *ServiceHandler) UpdateServiceEnvironments(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamID")
	projectID := chi.URLParam(r, "projectID")
	serviceID := chi.URLParam(r, "serviceID")

	var updateRequest updateServiceEnvironmentsRequest
	if err := json.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	if err := validator.New().Struct(updateRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	userID := r.Context().Value(middleware.UserIDKey).(string)

	tx, err := repository.StartTransaction(h.DB, r.Context())
	if err != nil {
		zap.L().Error("Failed to begin transaction", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
		return
	}
	defer repository.DeferRollback(tx, r.Context())

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

	service, err := repository.GetServiceByID(r.Context(), tx, serviceID, teamID, projectID)
	if err != nil {
		zap.L().Error("Failed to find service", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to find service", "FAILED_TO_FIND_SERVICE")
		return
	}
	if service == nil {
		response.RespondWithError(w, http.StatusNotFound, "Service not found", "SERVICE_NOT_FOUND")
		return
	}

	var environments []models.ServiceEnvironment
	for _, envReq := range updateRequest.Environments {
		env, err := repository.GetServiceEnvironment(r.Context(), tx, envReq.ID, serviceID)
		if err != nil {
			zap.L().Error("Failed to find environment variable", zap.Error(err), zap.Int64("envID", envReq.ID))
			response.RespondWithError(w, http.StatusNotFound, "Environment variable not found", "ENVIRONMENT_VARIABLE_NOT_FOUND")
			return
		}

		updateReq := updateServiceEnvironmentRequest{
			Key:   envReq.Key,
			Value: envReq.Value,
		}
		updatedEnv := updateEnvironmentFromRequest(*env, updateReq)
		environments = append(environments, updatedEnv)
	}

	if err := repository.UpdateServiceEnvironments(r.Context(), tx, environments); err != nil {
		zap.L().Error("Failed to update service environments", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to update service environments", "FAILED_TO_UPDATE_SERVICE_ENVIRONMENTS")
		return
	}

	repository.CommitTransaction(tx, r.Context())
	response.RespondWithJSON(w, http.StatusOK, environments)
}

// +----------------------------------------------+
// | Delete Service Environment                   |
// +----------------------------------------------+

// DeleteServiceEnvironment godoc
// @Summary Delete an environment variable for a service
// @Description Deletes an existing environment variable for a specific service within a team and project
// @Tags service
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param projectID path string true "Project ID"
// @Param serviceID path string true "Service ID"
// @Param envID path int true "Environment Variable ID"
// @Success 200 {object} response.SuccessResponse "Environment variable deleted successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request or team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 404 {object} response.ErrorResponse "Service or environment variable not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/projects/{projectID}/services/{serviceID}/env/{envID} [delete]
// @Security BearerAuth
func (h *ServiceHandler) DeleteServiceEnvironment(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamID")
	projectID := chi.URLParam(r, "projectID")
	serviceID := chi.URLParam(r, "serviceID")
	envIDStr := chi.URLParam(r, "envID")

	envID, err := strconv.ParseInt(envIDStr, 10, 64)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid environment variable ID", "INVALID_ENV_ID")
		return
	}

	userID := r.Context().Value(middleware.UserIDKey).(string)

	tx, err := repository.StartTransaction(h.DB, r.Context())
	if err != nil {
		zap.L().Error("Failed to begin transaction", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
		return
	}
	defer repository.DeferRollback(tx, r.Context())

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

	service, err := repository.GetServiceByID(r.Context(), tx, serviceID, teamID, projectID)
	if err != nil {
		zap.L().Error("Failed to find service", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to find service", "FAILED_TO_FIND_SERVICE")
		return
	}
	if service == nil {
		response.RespondWithError(w, http.StatusNotFound, "Service not found", "SERVICE_NOT_FOUND")
		return
	}

	if err := repository.DeleteServiceEnvironment(r.Context(), tx, envID, serviceID); err != nil {
		if err.Error() == "environment variable not found" {
			response.RespondWithError(w, http.StatusNotFound, "Environment variable not found", "ENVIRONMENT_VARIABLE_NOT_FOUND")
			return
		}
		zap.L().Error("Failed to delete service environment", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to delete service environment", "FAILED_TO_DELETE_SERVICE_ENVIRONMENT")
		return
	}

	repository.CommitTransaction(tx, r.Context())
	response.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Environment variable deleted successfully"})
}

// +----------------------------------------------+
// | Helper Functions                             |
// +----------------------------------------------+

func updateEnvironmentFromRequest(existingEnv models.ServiceEnvironment, updateRequest updateServiceEnvironmentRequest) models.ServiceEnvironment {
	if updateRequest.Key != nil {
		existingEnv.Key = *updateRequest.Key
	}
	if updateRequest.Value != nil {
		existingEnv.Value = *updateRequest.Value
	}
	existingEnv.UpdatedAt = time.Now()

	return existingEnv
}
