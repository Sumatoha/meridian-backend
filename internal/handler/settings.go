package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/meridian/api/internal/dto"
	"github.com/meridian/api/internal/repository"
)

type SettingsHandler struct {
	queries *repository.Queries
	logger  *slog.Logger
}

func NewSettingsHandler(queries *repository.Queries, logger *slog.Logger) *SettingsHandler {
	return &SettingsHandler{queries: queries, logger: logger}
}

func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	accountID, ok := parseUUID(chi.URLParam(r, "account_id"), h.logger)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_id", "invalid account ID")
		return
	}

	settings, err := h.queries.GetBrandSettings(r.Context(), accountID)
	if err != nil {
		respondError(w, http.StatusNotFound, "not_found", "brand settings not found")
		return
	}

	respondJSON(w, http.StatusOK, settingsToDTO(settings))
}

func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	accountID, ok := parseUUID(chi.URLParam(r, "account_id"), h.logger)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_id", "invalid account ID")
		return
	}

	var req dto.BrandSettingsDTO
	if err := parseJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	// Validate content mix sums to 100
	if req.MixUseful+req.MixSelling+req.MixPersonal+req.MixEntertaining != 100 {
		respondError(w, http.StatusBadRequest, "validation_error", "content mix percentages must sum to 100")
		return
	}

	teamJSON, _ := json.Marshal(req.TeamMembers)
	eventsJSON, _ := json.Marshal(req.UpcomingEvents)

	settings, err := h.queries.UpsertBrandSettings(r.Context(), repository.UpsertBrandSettingsParams{
		InstagramAccountID:  accountID,
		ContentGoal:         req.ContentGoal,
		ToneTraits:          req.ToneTraits,
		ToneCustomNote:      req.ToneCustomNote,
		MixUseful:           int32(req.MixUseful),
		MixSelling:          int32(req.MixSelling),
		MixPersonal:         int32(req.MixPersonal),
		MixEntertaining:     int32(req.MixEntertaining),
		FormatReelsEnabled:  req.FormatReelsEnabled,
		FormatReelsPct:      int32(req.FormatReelsPct),
		FormatCarouselEnabled: req.FormatCarouselEnabled,
		FormatCarouselPct:   int32(req.FormatCarouselPct),
		FormatPhotoEnabled:  req.FormatPhotoEnabled,
		FormatPhotoPct:      int32(req.FormatPhotoPct),
		BannedTopics:        req.BannedTopics,
		BannedWords:         req.BannedWords,
		CompetitorNames:     req.CompetitorNames,
		ContentRestrictions: req.ContentRestrictions,
		CustomRules:         req.CustomRules,
		ProductsServices:    req.ProductsServices,
		TargetAudience:      req.TargetAudience,
		Usp:                 req.USP,
		TeamMembers:         teamJSON,
		LocationAddress:     req.LocationAddress,
		WorkingHours:        req.WorkingHours,
		UpcomingEvents:      eventsJSON,
		ContentLanguage:     req.ContentLanguage,
		PostingFrequency:    req.PostingFrequency,
		Niche:               &req.Niche,
	})
	if err != nil {
		h.logger.Error("upsert settings failed", slog.String("error", err.Error()))
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to update settings")
		return
	}

	respondJSON(w, http.StatusOK, settingsToDTO(settings))
}

func settingsToDTO(s repository.BrandSetting) dto.BrandSettingsDTO {
	var teamMembers []dto.TeamMember
	if s.TeamMembers != nil {
		json.Unmarshal(s.TeamMembers, &teamMembers)
	}

	var events []dto.UpcomingEvent
	if s.UpcomingEvents != nil {
		json.Unmarshal(s.UpcomingEvents, &events)
	}

	niche := ""
	if s.Niche != nil {
		niche = *s.Niche
	}

	return dto.BrandSettingsDTO{
		ContentGoal:           s.ContentGoal,
		ToneTraits:            s.ToneTraits,
		ToneCustomNote:        s.ToneCustomNote,
		MixUseful:             int(s.MixUseful),
		MixSelling:            int(s.MixSelling),
		MixPersonal:           int(s.MixPersonal),
		MixEntertaining:       int(s.MixEntertaining),
		FormatReelsEnabled:    s.FormatReelsEnabled,
		FormatReelsPct:        int(s.FormatReelsPct),
		FormatCarouselEnabled: s.FormatCarouselEnabled,
		FormatCarouselPct:     int(s.FormatCarouselPct),
		FormatPhotoEnabled:    s.FormatPhotoEnabled,
		FormatPhotoPct:        int(s.FormatPhotoPct),
		BannedTopics:          s.BannedTopics,
		BannedWords:           s.BannedWords,
		CompetitorNames:       s.CompetitorNames,
		ContentRestrictions:   s.ContentRestrictions,
		CustomRules:           s.CustomRules,
		ProductsServices:      s.ProductsServices,
		TargetAudience:        s.TargetAudience,
		USP:                   s.Usp,
		TeamMembers:           teamMembers,
		LocationAddress:       s.LocationAddress,
		WorkingHours:          s.WorkingHours,
		UpcomingEvents:        events,
		ContentLanguage:       s.ContentLanguage,
		PostingFrequency:      s.PostingFrequency,
		Niche:                 niche,
	}
}
