package privatekey

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/response"
)

// +----------------------------------------------+
// | Get Private Keys                            |
// +----------------------------------------------+

// GetPrivateKeys godoc
// @Summary Get all private keys for a team
// @Description Retrieves all private keys associated with a specific team
// @Tags privatekey
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Success 200 {object} response.SuccessResponse{data=[]models.PrivateKey} "Private keys retrieved successfully"
// @Failure 400 {object} response.ErrorResponse "Team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /team/{teamID}/private-keys [get]
// @Security Bearer
func (h *PrivateKeyHandler) GetPrivateKeys(w http.ResponseWriter, r *http.Request) {
	// Get the team ID from the URL
	teamID := chi.URLParam(r, "teamID")

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

	// Check if user has access to the team
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

	// Get private keys for the team
	privateKeys, err := repository.GetPrivateKeysByTeamID(r.Context(), tx, teamID)
	if err != nil {
		zap.L().Error("Failed to get private keys", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get private keys", "FAILED_TO_GET_PRIVATE_KEYS")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Response
	response.RespondWithJSON(w, http.StatusOK, "Private keys retrieved successfully", privateKeys)
}

// +----------------------------------------------+
// | Get Private Key                             |
// +----------------------------------------------+

// GetPrivateKey godoc
// @Summary Get a specific private key
// @Description Retrieves a specific private key by ID within a team
// @Tags privatekey
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param privateKeyID path string true "Private Key ID"
// @Success 200 {object} response.SuccessResponse{data=models.PrivateKey} "Private key retrieved successfully"
// @Failure 400 {object} response.ErrorResponse "Team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 404 {object} response.ErrorResponse "Private key not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /team/{teamID}/private-keys/{privateKeyID} [get]
// @Security Bearer
func (h *PrivateKeyHandler) GetPrivateKey(w http.ResponseWriter, r *http.Request) {
	// Get the team ID and private key ID from the URL
	teamID := chi.URLParam(r, "teamID")
	privateKeyID := chi.URLParam(r, "privateKeyID")

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

	// Check if user has access to the team
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

	// Get the private key by ID
	privateKey, err := repository.GetPrivateKeyByID(r.Context(), tx, privateKeyID, teamID)
	if err != nil {
		zap.L().Error("Failed to get private key", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to get private key", "FAILED_TO_GET_PRIVATE_KEY")
		return
	}

	if privateKey == nil {
		response.RespondWithError(w, http.StatusNotFound, "Private key not found", "PRIVATE_KEY_NOT_FOUND")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Response
	response.RespondWithJSON(w, http.StatusOK, "Private key retrieved successfully", privateKey)
}
