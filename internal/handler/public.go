package handler

import (
	"context"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/meridian/api/internal/dto"
)

var igUsernameRegex = regexp.MustCompile(`^[a-zA-Z0-9._]{1,30}$`)

// PublicAnalyzer is the subset of AnalysisService needed by PublicHandler.
type PublicAnalyzer interface {
	AnalyzePublic(ctx context.Context, username string) (dto.PublicAuditResult, error)
}

type PublicHandler struct {
	analyzer PublicAnalyzer
	logger   *slog.Logger
}

func NewPublicHandler(analyzer PublicAnalyzer, logger *slog.Logger) *PublicHandler {
	return &PublicHandler{
		analyzer: analyzer,
		logger:   logger,
	}
}

func (h *PublicHandler) StartAudit(w http.ResponseWriter, r *http.Request) {
	var req dto.PublicAuditRequest
	if err := parseJSON(r, &req); err != nil {
		h.logger.Warn("audit: failed to parse request body",
			slog.String("error", err.Error()),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("content_type", r.Header.Get("Content-Type")),
		)
		respondError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	// Normalize: strip @, trim whitespace
	username := strings.TrimSpace(req.IGUsername)
	username = strings.TrimPrefix(username, "@")

	if username == "" {
		respondError(w, http.StatusBadRequest, "validation_error", "ig_username is required")
		return
	}

	if !igUsernameRegex.MatchString(username) {
		respondError(w, http.StatusBadRequest, "validation_error", "invalid Instagram username")
		return
	}

	h.logger.Info("audit: started",
		slog.String("ig_username", username),
		slog.String("remote_addr", r.RemoteAddr),
	)

	// Run synchronous analysis (rate-limited to 3/hour per IP at router level)
	result, err := h.analyzer.AnalyzePublic(r.Context(), username)
	if err != nil {
		h.logger.Error("audit: analysis failed",
			slog.String("ig_username", username),
			slog.String("error", err.Error()),
		)
		respondError(w, http.StatusUnprocessableEntity, "analysis_failed", "could not analyze this profile — it may be private or have no posts")
		return
	}

	result.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	respondJSON(w, http.StatusOK, map[string]any{
		"data": result,
	})
}

func (h *PublicHandler) GetAudit(w http.ResponseWriter, r *http.Request) {
	// Not used anymore — audit is synchronous
	respondError(w, http.StatusGone, "deprecated", "audit results are returned inline")
}
