package user

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/response"
)

// GetMe godoc
// @Summary Get current user information
// @Description Retrieves the profile information of the currently authenticated user
// @Tags user
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.User "User profile information"
// @Failure 401 {object} response.ErrorResponse "Unauthorized - invalid or missing token"
// @Failure 404 {object} response.ErrorResponse "User not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /user/me [get]
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	// Get userID from context (set by AuthRequiredMiddleware)
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		response.RespondWithError(w, http.StatusUnauthorized, "User not authenticated", "USER_NOT_AUTHENTICATED")
		return
	}

	// Begin the transaction
	tx, err := repository.StartTransaction(h.DB, r.Context())
	if err != nil {
		zap.L().Error("Failed to begin transaction", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to begin transaction", "FAILED_TO_BEGIN_TRANSACTION")
		return
	}
	defer repository.DeferRollback(tx, r.Context())

	// Get the user by ID
	user, err := repository.GetUserByID(r.Context(), tx, userID)
	if err != nil {
		zap.L().Error("Failed to get user by ID", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get user by ID", "FAILED_TO_GET_USER_BY_ID")
		return
	}

	// If the user is not found, return an error
	if user == nil {
		response.RespondWithError(w, http.StatusNotFound, "User not found", "USER_NOT_FOUND")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Remove password hash from response for security
	user.PasswordHash = nil

	// Respond with the user data
	response.RespondWithJSON(w, http.StatusOK, user)
}