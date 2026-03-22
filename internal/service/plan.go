package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/meridian/api/internal/ai"
	"github.com/meridian/api/internal/dto"
	"github.com/meridian/api/internal/repository"
)

type PlanService struct {
	queries  *repository.Queries
	aiClient *ai.Client
	logger   *slog.Logger
}

func NewPlanService(queries *repository.Queries, aiClient *ai.Client, logger *slog.Logger) *PlanService {
	return &PlanService{queries: queries, aiClient: aiClient, logger: logger}
}

// GeneratePlan creates a new content plan using AI.
func (s *PlanService) GeneratePlan(ctx context.Context, accountID uuid.UUID, startDate time.Time) (uuid.UUID, error) {
	// Load settings (use defaults if not configured yet)
	settings, err := s.queries.GetBrandSettings(ctx, accountID)
	if err != nil {
		s.logger.Warn("no brand settings found, using defaults for plan generation",
			slog.String("account_id", accountID.String()),
		)
		settings = repository.BrandSetting{
			ContentGoal:           "reach",
			ContentLanguage:       "ru",
			PostingFrequency:      "daily",
			MixUseful:             40,
			MixSelling:            25,
			MixPersonal:           20,
			MixEntertaining:       15,
			FormatReelsEnabled:    true,
			FormatReelsPct:        40,
			FormatCarouselEnabled: true,
			FormatCarouselPct:     30,
			FormatPhotoEnabled:    true,
			FormatPhotoPct:        30,
			ToneTraits:            []string{"friendly"},
			BannedTopics:          []string{},
			BannedWords:           []string{},
			CompetitorNames:       []string{},
			ContentRestrictions:   []string{},
		}
	}

	// Calculate end date (30 days) and total slots based on frequency
	endDate := startDate.AddDate(0, 0, 29)
	totalSlots := calculateTotalSlots(settings.PostingFrequency, 30)

	// Build system prompt from settings
	systemPrompt := s.buildPlanSystemPrompt(settings)

	// Generate slots in batches to avoid AI timeout
	// Each batch generates ~10 slots which takes ~1-2 min
	s.logger.Info("calling AI for plan generation",
		slog.String("account_id", accountID.String()),
		slog.Int("total_slots", totalSlots),
	)

	var allSlots []slotAIResponse
	batchSize := 10
	for batchStart := 0; batchStart < totalSlots; batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > totalSlots {
			batchEnd = totalSlots
		}
		batchCount := batchEnd - batchStart

		// Calculate dates for this batch
		batchStartDate := startDate.AddDate(0, 0, batchStart)
		batchEndDate := startDate.AddDate(0, 0, batchEnd-1)

		userPrompt := ai.BuildPlanUserPrompt(
			batchCount,
			batchStartDate.Format("2006-01-02"),
			batchEndDate.Format("2006-01-02"),
		)

		s.logger.Info("AI batch request",
			slog.Int("batch", batchStart/batchSize+1),
			slog.Int("slots", batchCount),
			slog.String("from", batchStartDate.Format("2006-01-02")),
			slog.String("to", batchEndDate.Format("2006-01-02")),
		)

		rawResponse, err := s.aiClient.Generate(ctx, systemPrompt, userPrompt, 16000)
		if err != nil {
			return uuid.Nil, fmt.Errorf("generate plan: AI batch %d: %w", batchStart/batchSize+1, err)
		}

		s.logger.Info("AI batch response received",
			slog.Int("batch", batchStart/batchSize+1),
			slog.Int("response_len", len(rawResponse)),
		)

		batchSlots, err := ai.ParseJSON[[]slotAIResponse](rawResponse)
		if err != nil {
			s.logger.Error("parse AI batch failed",
				slog.Int("batch", batchStart/batchSize+1),
				slog.String("error", err.Error()),
				slog.String("raw_response", truncateStr(rawResponse, 500)),
			)
			return uuid.Nil, fmt.Errorf("generate plan: parse batch %d: %w", batchStart/batchSize+1, err)
		}

		// Fix day numbers to be sequential across batches
		for i := range batchSlots {
			batchSlots[i].DayNumber = batchStart + i + 1
		}

		allSlots = append(allSlots, batchSlots...)
		s.logger.Info("batch parsed",
			slog.Int("batch", batchStart/batchSize+1),
			slog.Int("slots", len(batchSlots)),
			slog.Int("total_so_far", len(allSlots)),
		)
	}

	if len(allSlots) == 0 {
		return uuid.Nil, fmt.Errorf("generate plan: AI returned 0 slots")
	}

	slots := allSlots

	// NOW create plan record — only after AI succeeded
	title := fmt.Sprintf("%s %d Plan", startDate.Month().String(), startDate.Year())
	plan, err := s.queries.CreatePlan(ctx, repository.CreatePlanParams{
		InstagramAccountID: accountID,
		Title:              title,
		StartDate:          startDate,
		EndDate:            endDate,
		TotalSlots:         int32(len(slots)), // Use actual slot count from AI
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("generate plan: create plan: %w", err)
	}

	// Insert slots
	s.logger.Info("inserting slots",
		slog.String("plan_id", plan.ID.String()),
		slog.Int("ai_returned", len(slots)),
	)

	inserted := 0
	var lastErr error
	for i := range slots {
		slot := &slots[i]
		// Normalize AI output — DB has CHECK constraints for lowercase values
		slot.ContentType = strings.ToLower(strings.TrimSpace(slot.ContentType))
		slot.Format = strings.ToLower(strings.TrimSpace(slot.Format))
		slot.ScheduledDate = strings.TrimSpace(slot.ScheduledDate)
		slot.ScheduledTime = strings.TrimSpace(slot.ScheduledTime)

		// Validate content_type
		switch slot.ContentType {
		case "useful", "selling", "personal", "entertaining":
			// ok
		default:
			s.logger.Warn("invalid content_type from AI, defaulting to useful",
				slog.Int("day", slot.DayNumber),
				slog.String("raw", slot.ContentType),
			)
			slot.ContentType = "useful"
		}

		// Validate format
		switch slot.Format {
		case "reels", "carousel", "photo":
			// ok
		default:
			s.logger.Warn("invalid format from AI, defaulting to photo",
				slog.Int("day", slot.DayNumber),
				slog.String("raw", slot.Format),
			)
			slot.Format = "photo"
		}
	}

	for _, slot := range slots {
		briefJSON, err := json.Marshal(slot.Brief)
		if err != nil {
			s.logger.Error("marshal brief failed", slog.Int("day", slot.DayNumber), slog.String("error", err.Error()))
			lastErr = err
			continue
		}

		scheduledDate, err := time.Parse("2006-01-02", slot.ScheduledDate)
		if err != nil {
			s.logger.Error("parse scheduled_date failed",
				slog.Int("day", slot.DayNumber),
				slog.String("raw", slot.ScheduledDate),
				slog.String("error", err.Error()),
			)
			lastErr = err
			continue
		}

		scheduledTime, err := time.Parse("15:04", slot.ScheduledTime)
		if err != nil {
			s.logger.Error("parse scheduled_time failed",
				slog.Int("day", slot.DayNumber),
				slog.String("raw", slot.ScheduledTime),
				slog.String("error", err.Error()),
			)
			lastErr = err
			continue
		}

		hashtags := slot.Hashtags
		if hashtags == nil {
			hashtags = []string{}
		}

		_, err = s.queries.CreateSlot(ctx, repository.CreateSlotParams{
			PlanID:        plan.ID,
			DayNumber:     int32(slot.DayNumber),
			ScheduledDate: scheduledDate,
			ScheduledTime: scheduledTime,
			Title:         slot.Title,
			ContentType:   slot.ContentType,
			Format:        slot.Format,
			Brief:         briefJSON,
			Caption:       slot.Caption,
			Hashtags:      hashtags,
			Cta:           &slot.CTA,
		})
		if err != nil {
			s.logger.Error("insert slot failed",
				slog.Int("day", slot.DayNumber),
				slog.String("title", slot.Title),
				slog.String("content_type", slot.ContentType),
				slog.String("format", slot.Format),
				slog.String("error", err.Error()),
			)
			lastErr = err
			continue
		}
		inserted++
	}

	s.logger.Info("plan generated",
		slog.String("plan_id", plan.ID.String()),
		slog.Int("ai_returned", len(slots)),
		slog.Int("inserted", inserted),
	)

	if inserted == 0 && len(slots) > 0 {
		return uuid.Nil, fmt.Errorf("generate plan: all %d slots failed to insert: %w", len(slots), lastErr)
	}

	if inserted < len(slots) {
		s.logger.Warn("some slots failed to insert",
			slog.Int("failed", len(slots)-inserted),
			slog.Int("inserted", inserted),
		)
	}

	return plan.ID, nil
}

// GetPlan returns a plan with all its slots.
func (s *PlanService) GetPlan(ctx context.Context, planID uuid.UUID) (dto.ContentPlanDTO, error) {
	plan, err := s.queries.GetPlanByID(ctx, planID)
	if err != nil {
		return dto.ContentPlanDTO{}, fmt.Errorf("get plan: %w", err)
	}

	slots, err := s.queries.GetSlotsByPlanID(ctx, planID)
	if err != nil {
		return dto.ContentPlanDTO{}, fmt.Errorf("get plan slots: %w", err)
	}

	slotDTOs := make([]dto.ContentSlotDTO, 0, len(slots))
	for _, slot := range slots {
		d, err := slotToDTO(slot)
		if err != nil {
			continue
		}
		slotDTOs = append(slotDTOs, d)
	}

	return dto.ContentPlanDTO{
		ContentPlanSummaryDTO: planToSummaryDTO(plan),
		Slots:                 slotDTOs,
	}, nil
}

// ListPlans returns all plans for an account.
func (s *PlanService) ListPlans(ctx context.Context, accountID uuid.UUID) ([]dto.ContentPlanSummaryDTO, error) {
	plans, err := s.queries.GetPlansByAccountID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}

	result := make([]dto.ContentPlanSummaryDTO, 0, len(plans))
	for _, p := range plans {
		result = append(result, planToSummaryDTO(p))
	}
	return result, nil
}

// UpdatePlanStatus changes a plan's status.
func (s *PlanService) UpdatePlanStatus(ctx context.Context, planID uuid.UUID, status string) error {
	_, err := s.queries.UpdatePlanStatus(ctx, repository.UpdatePlanStatusParams{
		ID:     planID,
		Status: status,
	})
	if err != nil {
		return fmt.Errorf("update plan status: %w", err)
	}
	return nil
}

// DeletePlan removes a plan and its slots.
func (s *PlanService) DeletePlan(ctx context.Context, planID uuid.UUID) error {
	if err := s.queries.DeletePlan(ctx, planID); err != nil {
		return fmt.Errorf("delete plan: %w", err)
	}
	return nil
}

func (s *PlanService) buildPlanSystemPrompt(settings repository.BrandSetting) string {
	toneCustom := ""
	if settings.ToneCustomNote != nil {
		toneCustom = *settings.ToneCustomNote
	}

	formats := buildFormatsString(settings)
	customRules := ""
	if settings.CustomRules != nil {
		customRules = *settings.CustomRules
	}

	productsServices := derefStr(settings.ProductsServices)
	targetAudience := derefStr(settings.TargetAudience)
	usp := derefStr(settings.Usp)
	location := derefStr(settings.LocationAddress)
	hours := derefStr(settings.WorkingHours)

	teamJSON := "[]"
	if settings.TeamMembers != nil {
		teamJSON = string(settings.TeamMembers)
	}
	eventsJSON := "[]"
	if settings.UpcomingEvents != nil {
		eventsJSON = string(settings.UpcomingEvents)
	}

	return fmt.Sprintf(ai.PlanSystemPromptTemplate(),
		settings.ContentGoal,
		strings.Join(settings.ToneTraits, ", "), toneCustom,
		settings.MixUseful, settings.MixSelling, settings.MixPersonal, settings.MixEntertaining,
		formats,
		settings.PostingFrequency,
		strings.Join(settings.CompetitorNames, ", "),
		strings.Join(settings.BannedWords, ", "),
		strings.Join(settings.BannedTopics, ", "),
		strings.Join(settings.ContentRestrictions, ", "),
		customRules,
		productsServices,
		targetAudience,
		usp,
		teamJSON,
		location, hours,
		eventsJSON,
		settings.ContentLanguage,
	)
}

func buildFormatsString(s repository.BrandSetting) string {
	var parts []string
	if s.FormatReelsEnabled {
		parts = append(parts, fmt.Sprintf("Reels %d%%", s.FormatReelsPct))
	}
	if s.FormatCarouselEnabled {
		parts = append(parts, fmt.Sprintf("Carousel %d%%", s.FormatCarouselPct))
	}
	if s.FormatPhotoEnabled {
		parts = append(parts, fmt.Sprintf("Photo %d%%", s.FormatPhotoPct))
	}
	return strings.Join(parts, ", ")
}

func calculateTotalSlots(frequency string, days int) int {
	switch frequency {
	case "daily":
		return days
	case "every_other_day":
		return days / 2
	case "3_per_week":
		return (days / 7) * 3
	case "2_per_week":
		return (days / 7) * 2
	default:
		return days
	}
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

type slotAIResponse struct {
	DayNumber     int              `json:"day_number"`
	ScheduledDate string           `json:"scheduled_date"`
	ScheduledTime string           `json:"scheduled_time"`
	Title         string           `json:"title"`
	ContentType   string           `json:"content_type"`
	Format        string           `json:"format"`
	Brief         dto.ContentBrief `json:"brief"`
	Caption       string           `json:"caption"`
	Hashtags      []string         `json:"hashtags"`
	CTA           string           `json:"cta"`
}

func planToSummaryDTO(p repository.ContentPlan) dto.ContentPlanSummaryDTO {
	return dto.ContentPlanSummaryDTO{
		ID:             p.ID,
		Title:          p.Title,
		StartDate:      p.StartDate.Format("2006-01-02"),
		EndDate:        p.EndDate.Format("2006-01-02"),
		Status:         p.Status,
		TotalSlots:     int(p.TotalSlots),
		ApprovedSlots:  int(p.ApprovedSlots),
		PublishedSlots: int(p.PublishedSlots),
		CreatedAt:      p.CreatedAt,
	}
}

func slotToDTO(s repository.ContentSlot) (dto.ContentSlotDTO, error) {
	var brief dto.ContentBrief
	if err := json.Unmarshal(s.Brief, &brief); err != nil {
		return dto.ContentSlotDTO{}, fmt.Errorf("unmarshal brief: %w", err)
	}

	var media []dto.MediaItem
	if err := json.Unmarshal(s.Media, &media); err != nil {
		media = []dto.MediaItem{}
	}

	result := dto.ContentSlotDTO{
		ID:            s.ID,
		PlanID:        s.PlanID,
		DayNumber:     int(s.DayNumber),
		ScheduledDate: s.ScheduledDate.Format("2006-01-02"),
		ScheduledTime: s.ScheduledTime.Format("15:04"),
		Title:         s.Title,
		ContentType:   s.ContentType,
		Format:        s.Format,
		Brief:         brief,
		Caption:       s.Caption,
		Hashtags:      s.Hashtags,
		CTA:           s.Cta,
		Media:         media,
		Status:        s.Status,
		IsUserContent: s.IsUserContent,
		PublishedAt:   s.PublishedAt,
		IGPostURL:     s.IgPostUrl,
		RegenCount:    int(s.RegenerationCount),
	}

	return result, nil
}
