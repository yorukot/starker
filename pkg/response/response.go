package response

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse is the response for an error
type ErrorResponse struct {
	Message string `json:"message"`
	ErrCode string `json:"err_code"`
}

// SuccessResponse is the response for a success
type SuccessResponse struct {
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// RespondWithError responds with an error message
func RespondWithError(w http.ResponseWriter, statusCode int, message, errCode string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Message: message,
		ErrCode: errCode,
	})
}

// RespondWithJSON responds with a JSON object
func RespondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// RespondWithData responds with a JSON object
func RespondWithData(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}
