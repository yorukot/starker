package models

import "time"

// Project represents a project within a team
type Project struct {
	ID          string    `json:"id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`                            // Unique identifier for the project
	TeamID      string    `json:"team_id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`                       // Associated team ID
	Name        string    `json:"name" example:"My Web Application"`                                  // Project name
	Description *string   `json:"description,omitempty" example:"A web application built with React"` // Project description
	UpdatedAt   time.Time `json:"updated_at" example:"2023-01-01T12:00:00Z"`                          // Timestamp when the project was last updated
	CreatedAt   time.Time `json:"created_at" example:"2023-01-01T12:00:00Z"`                          // Timestamp when the project was created
}
