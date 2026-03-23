package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/meridian/api/internal/auth"
	"github.com/meridian/api/internal/dto"
	"github.com/meridian/api/internal/service"
)

type ExportHandler struct {
	exportSvc *service.ExportService
	logger    *slog.Logger
}

func NewExportHandler(exportSvc *service.ExportService, logger *slog.Logger) *ExportHandler {
	return &ExportHandler{exportSvc: exportSvc, logger: logger}
}

func (h *ExportHandler) Export(w http.ResponseWriter, r *http.Request) {
	planID, ok := parseUUID(chi.URLParam(r, "plan_id"), h.logger)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_id", "invalid plan ID")
		return
	}

	format := r.URL.Query().Get("format")
	if format != "xlsx" && format != "pdf" {
		respondError(w, http.StatusBadRequest, "invalid_format", "format must be xlsx or pdf")
		return
	}

	userID := auth.UserID(r.Context())

	var data []byte
	var filename string
	var err error

	switch format {
	case "xlsx":
		data, filename, err = h.exportSvc.ExportExcel(r.Context(), userID, planID)
	case "pdf":
		data, filename, err = h.exportSvc.ExportPDF(r.Context(), userID, planID)
	}

	if err != nil {
		var tierErr *service.TierError
		if errors.As(err, &tierErr) {
			respondJSON(w, http.StatusForbidden, dto.TierLimitError{
				Code:      "tier_limit",
				Message:   tierErr.Message,
				Feature:   tierErr.Feature,
				UpgradeTo: tierErr.UpgradeTo,
			})
			return
		}
		h.logger.Error("export failed", slog.String("error", err.Error()))
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to export plan")
		return
	}

	contentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	if format == "pdf" {
		contentType = "text/html; charset=utf-8"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
