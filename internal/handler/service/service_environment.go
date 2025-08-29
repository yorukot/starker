package service

import (
	"encoding/json"
	"net/http"
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
// | Update Service Environments (Batch)          |
// +----------------------------------------------+

type updateServiceEnvironmentItem struct {
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
	environments, err = repository.GetServiceEnvironments(r.Context(), tx, serviceID)
	if err != nil {
		zap.L().Error("Failed to get service environments", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get service environments", "FAILED_TO_GET_SERVICE_ENVIRONMENTS")
		return
	}

	// Create maps for efficient comparison
	existingEnvMap := make(map[string]models.ServiceEnvironment)
	for _, env := range environments {
		existingEnvMap[env.Key] = env
	}

	requestEnvMap := make(map[string]updateServiceEnvironmentItem)
	for _, item := range updateRequest.Environments {
		if item.Key != nil {
			requestEnvMap[*item.Key] = item
		}
	}

	// Delete environment variables that exist in DB but not in request
	for key, existingEnv := range existingEnvMap {
		if _, exists := requestEnvMap[key]; !exists {
			err := repository.DeleteServiceEnvironment(r.Context(), tx, existingEnv.ID, serviceID)
			if err != nil {
				zap.L().Error("Failed to delete service environment", zap.Error(err), zap.String("key", key))
				response.RespondWithError(w, http.StatusInternalServerError, "Failed to delete service environment", "FAILED_TO_DELETE_SERVICE_ENVIRONMENT")
				return
			}
		}
	}

	// Update existing environment variables if values are different
	for key, requestItem := range requestEnvMap {
		if existingEnv, exists := existingEnvMap[key]; exists {
			// Check if value needs updating
			if requestItem.Value != nil && *requestItem.Value != existingEnv.Value {
				existingEnv.Value = *requestItem.Value
				existingEnv.UpdatedAt = time.Now()

				err := repository.UpdateServiceEnvironment(r.Context(), tx, existingEnv)
				if err != nil {
					zap.L().Error("Failed to update service environment", zap.Error(err), zap.String("key", key))
					response.RespondWithError(w, http.StatusInternalServerError, "Failed to update service environment", "FAILED_TO_UPDATE_SERVICE_ENVIRONMENT")
					return
				}
			}
		}
	}

	// Create new environment variables that don't exist in DB
	for key, requestItem := range requestEnvMap {
		if _, exists := existingEnvMap[key]; !exists {
			if requestItem.Value != nil {
				newEnv := models.ServiceEnvironment{
					ServiceID: serviceID,
					Key:       key,
					Value:     *requestItem.Value,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				err := repository.CreateServiceEnvironment(r.Context(), tx, newEnv)
				if err != nil {
					zap.L().Error("Failed to create service environment", zap.Error(err), zap.String("key", key))
					response.RespondWithError(w, http.StatusInternalServerError, "Failed to create service environment", "FAILED_TO_CREATE_SERVICE_ENVIRONMENT")
					return
				}
			}
		}
	}

	// Get updated environments after all operations
	updatedEnvironments, err := repository.GetServiceEnvironments(r.Context(), tx, serviceID)
	if err != nil {
		zap.L().Error("Failed to get updated service environments", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get updated service environments", "FAILED_TO_GET_UPDATED_SERVICE_ENVIRONMENTS")
		return
	}

	repository.CommitTransaction(tx, r.Context())
	response.RespondWithJSON(w, http.StatusOK, updatedEnvironments)
}
