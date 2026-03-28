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
	tierSvc  *TierService
	logger   *slog.Logger
}

func NewPlanService(queries *repository.Queries, aiClient *ai.Client, tierSvc *TierService, logger *slog.Logger) *PlanService {
	return &PlanService{queries: queries, aiClient: aiClient, tierSvc: tierSvc, logger: logger}
}

// GeneratePlan creates a new content plan using AI.
// langOverride optionally overrides the content_language from brand settings.
// GeneratePlanOptions contains optional overrides for plan generation.
type GeneratePlanOptions struct {
	ContentLanguage  string
	PostingFrequency string
	ContentGoal      string
	MixUseful        *int
	MixSelling       *int
	MixPersonal      *int
	MixEntertaining  *int
	BrandContext     string
}

func (s *PlanService) GeneratePlan(ctx context.Context, userID, accountID uuid.UUID, startDate time.Time, opts *GeneratePlanOptions) (uuid.UUID, error) {
	// Check tier limit before expensive AI call
	if err := s.tierSvc.CheckPlanGeneration(ctx, userID); err != nil {
		return uuid.Nil, err
	}

	// Delete existing plans for this account (enforce 1 active plan per account)
	if err := s.queries.DeletePlansByAccountID(ctx, accountID); err != nil {
		s.logger.Error("failed to delete existing plans",
			slog.String("account_id", accountID.String()),
			slog.String("error", err.Error()),
		)
		return uuid.Nil, fmt.Errorf("generate plan: delete existing: %w", err)
	}

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

	// Apply overrides if provided
	if opts != nil {
		if opts.ContentLanguage != "" {
			settings.ContentLanguage = opts.ContentLanguage
		}
		if opts.PostingFrequency != "" {
			settings.PostingFrequency = opts.PostingFrequency
		}
		if opts.ContentGoal != "" {
			settings.ContentGoal = opts.ContentGoal
		}
		if opts.MixUseful != nil {
			settings.MixUseful = int32(*opts.MixUseful)
		}
		if opts.MixSelling != nil {
			settings.MixSelling = int32(*opts.MixSelling)
		}
		if opts.MixPersonal != nil {
			settings.MixPersonal = int32(*opts.MixPersonal)
		}
		if opts.MixEntertaining != nil {
			settings.MixEntertaining = int32(*opts.MixEntertaining)
		}
	}

	// Calculate end date (30 days) and total slots based on frequency
	endDate := startDate.AddDate(0, 0, 29)
	totalSlots := calculateTotalSlots(settings.PostingFrequency, 30)

	// Build system prompt from settings + optional brand context
	brandContext := ""
	if opts != nil && opts.BrandContext != "" {
		brandContext = opts.BrandContext
	}
	systemPrompt := s.buildPlanSystemPrompt(settings, brandContext)

	// Create plan record IMMEDIATELY with 'generating' status so frontend can track
	title := fmt.Sprintf("%s %d Plan", startDate.Month().String(), startDate.Year())
	plan, err := s.queries.CreateGeneratingPlan(ctx, repository.CreateGeneratingPlanParams{
		InstagramAccountID: accountID,
		Title:              title,
		StartDate:          startDate,
		EndDate:            endDate,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("generate plan: create generating plan: %w", err)
	}

	// Helper to mark plan as failed
	failPlan := func(errMsg string) {
		if failErr := s.queries.FailPlan(ctx, repository.FailPlanParams{
			ID:           plan.ID,
			ErrorMessage: &errMsg,
		}); failErr != nil {
			s.logger.Error("failed to mark plan as failed",
				slog.String("plan_id", plan.ID.String()),
				slog.String("error", failErr.Error()),
			)
		}
	}

	var slots []slotAIResponse

	// Small plans (<=12 slots): single-phase generation — 1 API call instead of 5
	// Large plans (>12 slots): two-phase (skeleton + parallel detail batches)
	const singlePhaseThreshold = 12

	if totalSlots <= singlePhaseThreshold {
		// ── Single-phase: generate full plan in one call ──
		s.logger.Info("single-phase generation",
			slog.String("plan_id", plan.ID.String()),
			slog.Int("total_slots", totalSlots),
		)

		userPrompt := ai.BuildPlanUserPrompt(
			totalSlots,
			startDate.Format("2006-01-02"),
			endDate.Format("2006-01-02"),
		)

		raw, err := s.aiClient.Generate(ctx, systemPrompt, userPrompt, 12000)
		if err != nil {
			failPlan(fmt.Sprintf("single-phase generation failed: %v", err))
			return uuid.Nil, fmt.Errorf("generate plan: single-phase: %w", err)
		}

		parsed, err := ai.ParseJSON[[]slotAIResponse](raw)
		if err != nil {
			s.logger.Error("parse single-phase failed",
				slog.String("error", err.Error()),
				slog.String("raw", truncateStr(raw, 500)),
			)
			failPlan(fmt.Sprintf("single-phase parse failed: %v", err))
			return uuid.Nil, fmt.Errorf("generate plan: parse single-phase: %w", err)
		}

		s.logger.Info("single-phase complete",
			slog.Int("slots", len(parsed)),
		)

		if len(parsed) == 0 {
			failPlan("single-phase returned 0 slots")
			return uuid.Nil, fmt.Errorf("generate plan: single-phase returned 0 slots")
		}

		slots = parsed
	} else {
		// ── Two-phase: skeleton + parallel detail batches ──

		// Phase 1: Generate skeleton (fast — ~3 sec)
		s.logger.Info("phase 1: generating skeleton",
			slog.String("plan_id", plan.ID.String()),
			slog.Int("total_slots", totalSlots),
		)

		skeletonPrompt := ai.BuildSkeletonUserPrompt(
			totalSlots,
			startDate.Format("2006-01-02"),
			endDate.Format("2006-01-02"),
		)

		skeletonRaw, err := s.aiClient.Generate(ctx, systemPrompt, skeletonPrompt, 2500)
		if err != nil {
			failPlan(fmt.Sprintf("skeleton generation failed: %v", err))
			return uuid.Nil, fmt.Errorf("generate plan: skeleton: %w", err)
		}

		type skeletonSlot struct {
			DayNumber     int    `json:"day_number"`
			ScheduledDate string `json:"scheduled_date"`
			ScheduledTime string `json:"scheduled_time"`
			Title         string `json:"title"`
			ContentType   string `json:"content_type"`
			Format        string `json:"format"`
		}

		skeleton, err := ai.ParseJSON[[]skeletonSlot](skeletonRaw)
		if err != nil {
			s.logger.Error("parse skeleton failed",
				slog.String("error", err.Error()),
				slog.String("raw", truncateStr(skeletonRaw, 500)),
			)
			failPlan(fmt.Sprintf("skeleton parse failed: %v", err))
			return uuid.Nil, fmt.Errorf("generate plan: parse skeleton: %w", err)
		}

		s.logger.Info("phase 1 complete",
			slog.Int("skeleton_slots", len(skeleton)),
		)

		if len(skeleton) == 0 {
			failPlan("skeleton returned 0 slots")
			return uuid.Nil, fmt.Errorf("generate plan: skeleton returned 0 slots")
		}

		// Phase 2: Generate details in parallel (~10-15 sec)
		type batchRange struct {
			from, to int
		}
		groupSize := (len(skeleton) + 3) / 4 // ceil division by 4
		var batches []batchRange
		for i := 0; i < len(skeleton); i += groupSize {
			end := i + groupSize
			if end > len(skeleton) {
				end = len(skeleton)
			}
			dayFrom := skeleton[i].DayNumber
			dayTo := skeleton[end-1].DayNumber
			batches = append(batches, batchRange{from: dayFrom, to: dayTo})
		}

		s.logger.Info("phase 2: generating details in parallel",
			slog.Int("batches", len(batches)),
		)

		skeletonJSON := skeletonRaw

		type batchResult struct {
			index int
			slots []slotAIResponse
			err   error
		}

		results := make(chan batchResult, len(batches))

		for i, b := range batches {
			go func(idx int, br batchRange) {
				detailPrompt := ai.BuildDetailsUserPrompt(br.from, br.to, skeletonJSON)
				raw, err := s.aiClient.Generate(ctx, systemPrompt, detailPrompt, 10000)
				if err != nil {
					results <- batchResult{index: idx, err: fmt.Errorf("batch %d: %w", idx+1, err)}
					return
				}

				parsed, err := ai.ParseJSON[[]slotAIResponse](raw)
				if err != nil {
					s.logger.Error("parse detail batch failed",
						slog.Int("batch", idx+1),
						slog.String("error", err.Error()),
						slog.String("raw", truncateStr(raw, 500)),
					)
					results <- batchResult{index: idx, err: fmt.Errorf("parse batch %d: %w", idx+1, err)}
					return
				}

				s.logger.Info("detail batch complete",
					slog.Int("batch", idx+1),
					slog.Int("slots", len(parsed)),
				)
				results <- batchResult{index: idx, slots: parsed}
			}(i, b)
		}

		// Collect results
		allBatches := make([][]slotAIResponse, len(batches))
		for range batches {
			res := <-results
			if res.err != nil {
				failPlan(fmt.Sprintf("detail generation failed: %v", res.err))
				return uuid.Nil, fmt.Errorf("generate plan: details: %w", res.err)
			}
			allBatches[res.index] = res.slots
		}

		// Merge all slots in order
		for _, batch := range allBatches {
			slots = append(slots, batch...)
		}

		s.logger.Info("phase 2 complete",
			slog.Int("total_slots", len(slots)),
		)

		if len(slots) == 0 {
			failPlan("details returned 0 slots")
			return uuid.Nil, fmt.Errorf("generate plan: details returned 0 slots")
		}
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
		slot.ContentType = strings.ToLower(strings.TrimSpace(slot.ContentType))
		slot.Format = strings.ToLower(strings.TrimSpace(slot.Format))
		slot.ScheduledDate = strings.TrimSpace(slot.ScheduledDate)
		slot.ScheduledTime = strings.TrimSpace(slot.ScheduledTime)

		switch slot.ContentType {
		case "useful", "selling", "personal", "entertaining":
		default:
			s.logger.Warn("invalid content_type from AI, defaulting to useful",
				slog.Int("day", slot.DayNumber),
				slog.String("raw", slot.ContentType),
			)
			slot.ContentType = "useful"
		}

		switch slot.Format {
		case "reels", "carousel", "photo":
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
		failPlan(fmt.Sprintf("all %d slots failed to insert", len(slots)))
		return uuid.Nil, fmt.Errorf("generate plan: all %d slots failed to insert: %w", len(slots), lastErr)
	}

	if inserted < len(slots) {
		s.logger.Warn("some slots failed to insert",
			slog.Int("failed", len(slots)-inserted),
			slog.Int("inserted", inserted),
		)
	}

	// Finalize plan — set status to 'draft' with actual slot count
	if _, err := s.queries.FinalizePlan(ctx, repository.FinalizePlanParams{
		ID:         plan.ID,
		TotalSlots: int32(inserted),
	}); err != nil {
		s.logger.Error("failed to finalize plan",
			slog.String("plan_id", plan.ID.String()),
			slog.String("error", err.Error()),
		)
	}

	// Increment monthly usage counter after successful generation
	if err := s.tierSvc.IncrementPlanGeneration(ctx, userID); err != nil {
		s.logger.Error("failed to increment plan generation usage",
			slog.String("user_id", userID.String()),
			slog.String("error", err.Error()),
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

func (s *PlanService) buildPlanSystemPrompt(settings repository.BrandSetting, brandContext string) string {
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

	prompt := fmt.Sprintf(ai.PlanSystemPromptTemplate(),
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

	if brandContext != "" {
		prompt += fmt.Sprintf("\n\nADDITIONAL CONTEXT FROM USER (incorporate into the plan):\n%s", brandContext)
	}

	return prompt
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
		ErrorMessage:   p.ErrorMessage,
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
