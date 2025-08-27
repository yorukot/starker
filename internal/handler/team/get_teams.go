package team

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/response"
)

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
