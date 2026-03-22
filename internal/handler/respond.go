package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/meridian/api/internal/dto"
)

type dataEnvelope struct {
	Data interface{} `json:"data"`
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(dataEnvelope{Data: data})
	}
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, dto.ErrorResponse{
		Error: dto.ErrorDetail{Code: code, Message: message},
	})
}

func parseJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func parseUUID(s string, logger *slog.Logger) (uuid.UUID, bool) {
	id, err := uuid.Parse(s)
	if err != nil {
		logger.Warn("invalid UUID", slog.String("value", s))
		return uuid.Nil, false
	}
	return id, true
}
