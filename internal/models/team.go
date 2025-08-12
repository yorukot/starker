package models

import "time"

type Team struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"owner_id"`
	Name      string    `json:"name"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}

type TeamUser struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"team_id"`
	UserID    string    `json:"user_id"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}

type TeamInvite struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"team_id"`
	InvitedBy string    `json:"invited_by"`
	InvitedTo string    `json:"invited_to"`
	CreatedAt time.Time `json:"created_at"`
}

