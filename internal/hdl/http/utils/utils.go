package utils

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Data any `json:"data"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func SuccessPaginatedResponse(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func SuccessResponse(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(
		&Response{
			Data: data,
		},
	)
}

func ErrResponse(w http.ResponseWriter, statusCode int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(
		&ErrorResponse{
			Error: err.Error(),
		},
	)
}
