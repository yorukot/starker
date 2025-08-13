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
// | Create Private Key                          |
// +----------------------------------------------+

// TODO: Need to encrypt the private key before storing it in the database

// CreatePrivateKey godoc
// @Summary Create a new private key
// @Description Creates a new private key for SSH authentication within a team
// @Tags privatekey
// @Accept json
// @Produce json
// @Param teamID path string true "Team ID"
// @Param request body privatekeysvc.CreatePrivateKeyRequest true "Private key creation request"
// @Success 200 {object} response.SuccessResponse{data=models.PrivateKey} "Private key created successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body or team access denied"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /teams/{teamID}/private-keys [post]
// @Security BearerAuth
func (h *PrivateKeyHandler) CreatePrivateKey(w http.ResponseWriter, r *http.Request) {
	// Get the team ID from the URL
	teamID := chi.URLParam(r, "teamID")

	// Get the private key from the request body
	var createPrivateKeyRequest privatekeysvc.CreatePrivateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&createPrivateKeyRequest); err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST_BODY")
		return
	}

	// Validate the private key
	if err := privatekeysvc.PrivateKeyValidate(createPrivateKeyRequest); err != nil {
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

	// Generate the private key
	privateKey := privatekeysvc.GeneratePrivateKey(createPrivateKeyRequest, teamID)

	// Create the private key
	if err = repository.CreatePrivateKey(r.Context(), tx, privateKey); err != nil {
		zap.L().Error("Failed to create private key", zap.Error(err))
		response.RespondWithError(w, http.StatusInternalServerError, "Failed to create private key", "FAILED_TO_CREATE_PRIVATE_KEY")
		return
	}

	// Commit the transaction
	repository.CommitTransaction(tx, r.Context())

	// Response
	response.RespondWithJSON(w, http.StatusOK, "Private key created successfully", privateKey)
}
