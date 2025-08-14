package projectsvc

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/segmentio/ksuid"

	"github.com/yorukot/starker/internal/models"
)

type CreateProjectRequest struct {
	Name        string  `json:"name" validate:"required,min=3,max=255"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=500"`
}

type UpdateProjectRequest struct {
	Name        string  `json:"name" validate:"required,min=3,max=255"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=500"`
}

// ProjectValidate validates the create project request
func ProjectValidate(createProjectRequest CreateProjectRequest) error {
	return validator.New().Struct(createProjectRequest)
}

// ProjectUpdateValidate validates the update project request
func ProjectUpdateValidate(updateProjectRequest UpdateProjectRequest) error {
	return validator.New().Struct(updateProjectRequest)
}

// GenerateProject generates a project model for the create request
func GenerateProject(createProjectRequest CreateProjectRequest, teamID string) models.Project {
	now := time.Now()

	return models.Project{
		ID:          ksuid.New().String(),
		TeamID:      teamID,
		Name:        createProjectRequest.Name,
		Description: createProjectRequest.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// UpdateProjectFromRequest updates a project model with new values from update request
func UpdateProjectFromRequest(existingProject models.Project, updateProjectRequest UpdateProjectRequest) models.Project {
	existingProject.Name = updateProjectRequest.Name
	existingProject.Description = updateProjectRequest.Description
	existingProject.UpdatedAt = time.Now()

	return existingProject
}
