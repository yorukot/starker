package models

type ErrorResponse struct {
	Message string `json:"message"`
	ErrCode string `json:"err_code"`
}

type SuccessResponse struct {
	Message string `json:"message"`
	Data    any    `json:"data"`
}
