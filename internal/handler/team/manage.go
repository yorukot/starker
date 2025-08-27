package team

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/internal/service/teamsvc"
	"github.com/yorukot/starker/pkg/response"
)

// +----------------------------------------------+
// | Create Team                                  |
// +----------------------------------------------+

// CreateTeam godoc
// @Summary Create a new team
// @Description Creates a new team with the authenticated user as the owner
// @Tags team
// @Accept json
// @Produce json
// @Param request body teamsvc.CreateTeamRequest true "Create team request"
// @Success 200 {object} response.SuccessResponse "Team created successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams [post]
// @Security BearerAuth
func (h *TeamHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	// Decode the request body
	var createTeamRequest teamsvc.CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&createTeamRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	// Validate the request body
	if err := teamsvc.TeamValidate(createTeamRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	// Get user ID from context
	userID := r.Context().Value(middleware.UserIDKey).(string)

	// Begin the transaction
	tx, err := repository.StartTransaction(h.DB, r.Context())
	if err != nil {
		zap.L().Error("Failed to begin transaction", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
		return
	}

	defer repository.DeferRollback(tx, r.Context())

	// Generate the team and team user
	team, teamUser := teamsvc.GenerateTeam(createTeamRequest, userID)

	// Create the team and team user in the database
	if err = repository.CreateTeamAndTeamUser(r.Context(), tx, team, teamUser); err != nil {
		zap.L().Error("Failed to create team", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to create team", "FAILED_TO_CREATE_TEAM")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Respond with the team object
	response.RespondWithJSON(w, http.StatusOK, team)
}

// +----------------------------------------------+
// | Get Teams                                    |
// +----------------------------------------------+

// GetTeams godoc
// @Summary Get user's teams
// @Description Gets all teams that the authenticated user is a member of
// @Tags team
// @Accept json
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=[]models.Team} "Teams retrieved successfully"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams [get]
// @Security BearerAuth
func (h *TeamHandler) GetTeams(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID := r.Context().Value(middleware.UserIDKey).(string)

	// Begin the transaction
	tx, err := repository.StartTransaction(h.DB, r.Context())
	if err != nil {
		zap.L().Error("Failed to begin transaction", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
		return
	}

	defer repository.DeferRollback(tx, r.Context())

	// Get all the teams that the user is a member of
	teams, err := repository.GetTeamsByUserID(r.Context(), tx, userID)
	if err != nil {
		zap.L().Error("Failed to get teams by user ID", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get teams", "FAILED_TO_GET_TEAMS")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Return all the teams that the user is a member of
	response.RespondWithJSON(w, http.StatusOK, teams)
}

// +----------------------------------------------+
// | Get Team                                     |
// +----------------------------------------------+

// GetTeam godoc
// @Summary Get a specific team
// @Description Gets a specific team by ID if the authenticated user is a member of it
// @Tags team
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Success 200 {object} response.SuccessResponse{data=models.Team} "Team retrieved successfully"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 404 {object} response.ErrorResponse "Team not found or user is not a member"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID} [get]
// @Security BearerAuth
func (h *TeamHandler) GetTeam(w http.ResponseWriter, r *http.Request) {
	// Get the team ID from the URL
	teamID := chi.URLParam(r, "teamID")
	if teamID == "" {
		response.RespondWithError(w, http.StatusBadRequest, "Team ID is required", "TEAM_ID_REQUIRED")
		return
	}

	// Get user ID from context
	userID := r.Context().Value(middleware.UserIDKey).(string)

	// Begin the transaction
	tx, err := repository.StartTransaction(h.DB, r.Context())
	if err != nil {
		zap.L().Error("Failed to begin transaction", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
		return
	}

	defer repository.DeferRollback(tx, r.Context())

	// Get the team from the database (Will also check if the user is a member of the team in sql)
	// We doing this by use join so first we get the data from the team_users and then join the team data.
	team, err := repository.GetTeamByIDAndUserID(r.Context(), tx, teamID, userID)
	if err != nil {
		zap.L().Error("Failed to get team by ID and user ID", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get team", "FAILED_TO_GET_TEAM")
		return
	}

	// If we find the team, we return the team object, otherwise we return a 404
	if team == nil {
		response.RespondWithError(w, http.StatusNotFound, "Team not found or you are not a member", "TEAM_NOT_FOUND")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Return the team object
	response.RespondWithJSON(w, http.StatusOK, team)
}

// +----------------------------------------------+
// | Delete Team                                  |
// +----------------------------------------------+

// DeleteTeam godoc
// @Summary Delete a team
// @Description Deletes a team if the authenticated user is the owner
// @Tags team
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Success 200 {object} response.SuccessResponse "Team deleted successfully"
// @Failure 400 {object} response.ErrorResponse "Team ID is required"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 403 {object} response.ErrorResponse "Only team owner can delete the team"
// @Failure 404 {object} response.ErrorResponse "Team not found or user is not a member"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID} [delete]
// @Security BearerAuth
func (h *TeamHandler) DeleteTeam(w http.ResponseWriter, r *http.Request) {
	// Get the team ID from the URL
	teamID := chi.URLParam(r, "teamID")
	if teamID == "" {
		response.RespondWithError(w, http.StatusBadRequest, "Team ID is required", "TEAM_ID_REQUIRED")
		return
	}

	// Get user ID from context
	userID := r.Context().Value(middleware.UserIDKey).(string)

	// Begin the transaction
	tx, err := repository.StartTransaction(h.DB, r.Context())
	if err != nil {
		zap.L().Error("Failed to begin transaction", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
		return
	}

	defer repository.DeferRollback(tx, r.Context())

	// Get the team to check if user is a member and get owner info
	team, err := repository.GetTeamByIDAndUserID(r.Context(), tx, teamID, userID)
	if err != nil {
		zap.L().Error("Failed to get team by ID and user ID", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get team", "FAILED_TO_GET_TEAM")
		return
	}

	// Check if team exists and user is a member
	if team == nil {
		response.RespondWithError(w, http.StatusNotFound, "Team not found or you are not a member", "TEAM_NOT_FOUND")
		return
	}

	// Check if the user is the owner of the team
	if team.OwnerID != userID {
		response.RespondWithError(w, http.StatusForbidden, "Only team owner can delete the team", "NOT_TEAM_OWNER")
		return
	}

	// Delete the team and all related team users
	if err = repository.DeleteTeamAndAllRelatedData(r.Context(), tx, teamID); err != nil {
		zap.L().Error("Failed to delete team", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to delete team", "FAILED_TO_DELETE_TEAM")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Response
	response.RespondWithJSON(w, http.StatusOK, nil)
}
