package privatekey

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/internal/service/privatekeysvc"
	"github.com/yorukot/starker/pkg/response"
)

// +----------------------------------------------+
// | Update Private Key                          |
// +----------------------------------------------+

// UpdatePrivateKey godoc
// @Summary Update a private key
// @Description Updates a specific private key by ID within a team
// @Tags privatekey
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param privateKeyID path string true "Private Key ID"
// @Param request body privatekeysvc.UpdatePrivateKeyRequest true "Private key update request"
// @Success 200 {object} response.SuccessResponse{data=models.PrivateKey} "Private key updated successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body or team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 404 {object} response.ErrorResponse "Private key not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/private-keys/{privateKeyID} [patch]
// @Security BearerAuth
func (h *PrivateKeyHandler) UpdatePrivateKey(w http.ResponseWriter, r *http.Request) {
	// Get the team ID and private key ID from the URL
	teamID := chi.URLParam(r, "teamID")
	privateKeyID := chi.URLParam(r, "privateKeyID")

	// Get the update request from the request body
	var updatePrivateKeyRequest privatekeysvc.UpdatePrivateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&updatePrivateKeyRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	// Validate the update request
	if err := privatekeysvc.UpdatePrivateKeyValidate(updatePrivateKeyRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

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

	// Get the existing private key
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

	// Update the private key fields
	privatekeysvc.UpdatePrivateKeyFields(privateKey, updatePrivateKeyRequest)

	// Update the private key in the database
	if err = repository.UpdatePrivateKeyByID(r.Context(), tx, *privateKey); err != nil {
		zap.L().Error("Failed to update private key", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to update private key", "FAILED_TO_UPDATE_PRIVATE_KEY")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Response
	response.RespondWithJSON(w, http.StatusOK, privateKey)
}