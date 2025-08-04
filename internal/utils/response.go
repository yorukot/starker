package utils

import (
	"encoding/json"
	"net/http"

	"github.com/yorukot/stargo/internal/models"
)

// RespondWithError responds with an error message
func RespondWithError(w http.ResponseWriter, statusCode int, message, errCode string) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(models.ErrorResponse{
		Message: message,
		ErrCode: errCode,
	})
}

// RespondWithJSON responds with a JSON object
func RespondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
