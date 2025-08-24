package servicesvc

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/segmentio/ksuid"

	"github.com/yorukot/starker/internal/models"
)

type CreateServiceRequest struct {
	Name        string  `json:"name" validate:"required,min=3,max=255"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=500"`
	Type        string  `json:"type" validate:"required,oneof=docker compose"`
	ServerID    string  `json:"server_id" validate:"required"`
	ComposeFile string  `json:"compose_file" validate:"required"`
}

type UpdateServiceRequest struct {
	Name        *string              `json:"name,omitempty" validate:"omitempty,min=3,max=255"`
	Description *string              `json:"description,omitempty" validate:"omitempty,max=500"`
	Type        *string              `json:"type,omitempty" validate:"omitempty,oneof=docker compose"`
	State       *models.ServiceState `json:"status,omitempty" validate:"omitempty,oneof=running stopped starting stopping"`
}

// ServiceValidate validates the create service request
func ServiceValidate(createServiceRequest CreateServiceRequest) error {
	return validator.New().Struct(createServiceRequest)
}

// ServiceUpdateValidate validates the update service request
func ServiceUpdateValidate(updateServiceRequest UpdateServiceRequest) error {
	return validator.New().Struct(updateServiceRequest)
}

// GenerateService generates a service model for the create request
func GenerateService(createServiceRequest CreateServiceRequest, teamID, serverID, projectID string) models.Service {
	now := time.Now()

	return models.Service{
		ID:          ksuid.New().String(),
		TeamID:      teamID,
		ServerID:    serverID,
		ProjectID:   projectID,
		Name:        createServiceRequest.Name,
		Description: createServiceRequest.Description,
		Type:        createServiceRequest.Type,
		State:       models.ServiceStateStopped, // Default status
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// GenerateServiceComposeConfig generates a compose config model
func GenerateServiceComposeConfig(serviceID, composeFile string) models.ServiceComposeConfig {
	now := time.Now()

	return models.ServiceComposeConfig{
		ID:          ksuid.New().String(),
		ServiceID:   serviceID,
		ComposeFile: composeFile,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// UpdateServiceFromRequest updates a service model with new values from update request
func UpdateServiceFromRequest(existingService models.Service, updateServiceRequest UpdateServiceRequest) models.Service {
	if updateServiceRequest.Name != nil {
		existingService.Name = *updateServiceRequest.Name
	}
	if updateServiceRequest.Description != nil {
		existingService.Description = updateServiceRequest.Description
	}
	if updateServiceRequest.Type != nil {
		existingService.Type = *updateServiceRequest.Type
	}
	if updateServiceRequest.State != nil {
		existingService.State = *updateServiceRequest.State
	}
	existingService.UpdatedAt = time.Now()

	return existingService
}
