package dto

import (
	"time"

	"github.com/google/uuid"
)

// --- Error ---

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// --- Accounts ---

type CreateAccountRequest struct {
	IGUsername string `json:"ig_username"`
}

type OAuthURLResponse struct {
	URL string `json:"url"`
}

type OAuthCallbackRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

type OAuthCallbackResponse struct {
	Account AccountResponse `json:"account"`
	IsNew   bool            `json:"is_new"`
}

type AccountResponse struct {
	ID               uuid.UUID `json:"id"`
	IGUsername        string    `json:"ig_username"`
	IGUserID         *string   `json:"ig_user_id,omitempty"`
	ProfilePicURL    *string   `json:"profile_pic_url,omitempty"`
	FollowersCount   *int32    `json:"followers_count,omitempty"`
	IsOAuthConnected bool      `json:"is_oauth_connected"`
}

// --- Brand Settings ---

type TeamMember struct {
	Name    string `json:"name"`
	Role    string `json:"role"`
	FunFact string `json:"fun_fact,omitempty"`
}

type UpcomingEvent struct {
	Date        string `json:"date"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type BrandSettingsDTO struct {
	ContentGoal     string   `json:"content_goal"`
	ToneTraits      []string `json:"tone_traits"`
	ToneCustomNote  *string  `json:"tone_custom_note,omitempty"`
	MixUseful       int      `json:"mix_useful"`
	MixSelling      int      `json:"mix_selling"`
	MixPersonal     int      `json:"mix_personal"`
	MixEntertaining int      `json:"mix_entertaining"`

	FormatReelsEnabled    bool `json:"format_reels_enabled"`
	FormatReelsPct        int  `json:"format_reels_pct"`
	FormatCarouselEnabled bool `json:"format_carousel_enabled"`
	FormatCarouselPct     int  `json:"format_carousel_pct"`
	FormatPhotoEnabled    bool `json:"format_photo_enabled"`
	FormatPhotoPct        int  `json:"format_photo_pct"`

	BannedTopics        []string `json:"banned_topics"`
	BannedWords         []string `json:"banned_words"`
	CompetitorNames     []string `json:"competitor_names"`
	ContentRestrictions []string `json:"content_restrictions"`
	CustomRules         *string  `json:"custom_rules,omitempty"`

	ProductsServices *string         `json:"products_services,omitempty"`
	TargetAudience   *string         `json:"target_audience,omitempty"`
	USP              *string         `json:"usp,omitempty"`
	TeamMembers      []TeamMember    `json:"team_members,omitempty"`
	LocationAddress  *string         `json:"location_address,omitempty"`
	WorkingHours     *string         `json:"working_hours,omitempty"`
	UpcomingEvents   []UpcomingEvent `json:"upcoming_events,omitempty"`

	ContentLanguage  string `json:"content_language"`
	PostingFrequency string `json:"posting_frequency"`
	Niche            string `json:"niche,omitempty"`
}

// --- Brand DNA ---

type Strength struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type Recommendation struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
}

type BrandDnaDTO struct {
	ID                  uuid.UUID        `json:"id"`
	Score               int              `json:"score"`
	Tone                string           `json:"tone"`
	VisualStyle         *string          `json:"visual_style,omitempty"`
	StrongTopics        []string         `json:"strong_topics"`
	WeakAreas           []string         `json:"weak_areas"`
	BestFormats         []string         `json:"best_formats"`
	BestPostingTimes    []string         `json:"best_posting_times"`
	Strengths           []Strength       `json:"strengths"`
	Recommendations     []Recommendation `json:"recommendations"`
	CreatedAt           time.Time        `json:"created_at"`
}

// --- Analysis Status ---

type AnalysisStatusResponse struct {
	Status   string `json:"status"`
	Progress int    `json:"progress"`
}

type JobResponse struct {
	JobID uuid.UUID `json:"job_id"`
}

// --- Content Plans ---

type GeneratePlanRequest struct {
	StartDate        *string `json:"start_date,omitempty"`
	ContentLanguage  *string `json:"content_language,omitempty"`
	PostingFrequency *string `json:"posting_frequency,omitempty"`
	ContentGoal      *string `json:"content_goal,omitempty"`
	MixUseful        *int    `json:"mix_useful,omitempty"`
	MixSelling       *int    `json:"mix_selling,omitempty"`
	MixPersonal      *int    `json:"mix_personal,omitempty"`
	MixEntertaining  *int    `json:"mix_entertaining,omitempty"`
	BrandContext     *string `json:"brand_context,omitempty"`
}

type ContentPlanSummaryDTO struct {
	ID              uuid.UUID `json:"id"`
	Title           string    `json:"title"`
	StartDate       string    `json:"start_date"`
	EndDate         string    `json:"end_date"`
	Status          string    `json:"status"`
	TotalSlots      int       `json:"total_slots"`
	ApprovedSlots   int       `json:"approved_slots"`
	PublishedSlots  int       `json:"published_slots"`
	CreatedAt       time.Time `json:"created_at"`
}

type ContentPlanDTO struct {
	ContentPlanSummaryDTO
	Slots []ContentSlotDTO `json:"slots"`
}

type UpdatePlanRequest struct {
	Status string `json:"status"`
}

type GenerationStatusResponse struct {
	Status         string `json:"status"`
	SlotsGenerated int    `json:"slots_generated"`
	TotalSlots     int    `json:"total_slots"`
}

// --- Content Slots ---

type SceneDescription struct {
	Scene        int    `json:"scene"`
	Description  string `json:"description"`
	OnScreenText string `json:"on_screen_text,omitempty"`
	Duration     string `json:"duration,omitempty"`
}

type ContentBrief struct {
	VisualDescription string             `json:"visual_description"`
	SceneByScene      []SceneDescription `json:"scene_by_scene,omitempty"`
	Mood              string             `json:"mood,omitempty"`
	PhotoDirection    string             `json:"photo_direction,omitempty"`
	PeopleInFrame     string             `json:"people_in_frame,omitempty"`
	PropsNeeded       []string           `json:"props_needed,omitempty"`
	AspectRatio       string             `json:"aspect_ratio,omitempty"`
}

type MediaItem struct {
	StoragePath      string `json:"storage_path"`
	Type             string `json:"type"`
	Order            int    `json:"order"`
	OriginalFilename string `json:"original_filename,omitempty"`
	URL              string `json:"url,omitempty"`
}

type ContentSlotDTO struct {
	ID            uuid.UUID    `json:"id"`
	PlanID        uuid.UUID    `json:"plan_id"`
	DayNumber     int          `json:"day_number"`
	ScheduledDate string       `json:"scheduled_date"`
	ScheduledTime string       `json:"scheduled_time"`
	Title         string       `json:"title"`
	ContentType   string       `json:"content_type"`
	Format        string       `json:"format"`
	Brief         ContentBrief `json:"brief"`
	Caption       string       `json:"caption"`
	Hashtags      []string     `json:"hashtags"`
	CTA           *string      `json:"cta,omitempty"`
	Media         []MediaItem  `json:"media"`
	Status        string       `json:"status"`
	IsUserContent bool         `json:"is_user_content"`
	PublishedAt   *time.Time   `json:"published_at,omitempty"`
	IGPostURL     *string      `json:"ig_post_url,omitempty"`
	RegenCount    int          `json:"regeneration_count"`
}

type UpdateSlotRequest struct {
	Caption       *string  `json:"caption,omitempty"`
	Hashtags      []string `json:"hashtags,omitempty"`
	ScheduledTime *string  `json:"scheduled_time,omitempty"`
	ScheduledDate *string  `json:"scheduled_date,omitempty"`
	Status        *string  `json:"status,omitempty"`
	IsUserContent *bool    `json:"is_user_content,omitempty"`
}

type ApproveAllResponse struct {
	ApprovedCount    int `json:"approved_count"`
	SkippedCount     int `json:"skipped_count"`
	MissingMediaCount int `json:"missing_media_count"`
}

type StartPostingResponse struct {
	QueuedCount int `json:"queued_count"`
}

type MediaUploadResponse struct {
	Media []MediaItem `json:"media"`
}

// --- Tier ---

type TierInfoResponse struct {
	Plan                 string `json:"plan"`
	MaxAccounts          int    `json:"max_accounts"`
	AccountsUsed         int    `json:"accounts_used"`
	PlanGenerationsLimit int    `json:"plan_generations_limit"` // -1 = unlimited
	PlanGenerationsUsed  int    `json:"plan_generations_used"`
	AutoPosting          bool   `json:"auto_posting"`
	Export               bool   `json:"export"`
	Sharing              bool   `json:"sharing"`
	PeriodResetsAt       string `json:"period_resets_at"`
}

type TierLimitError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Feature   string `json:"feature"`
	Limit     int    `json:"limit"`
	Used      int    `json:"used"`
	UpgradeTo string `json:"upgrade_to,omitempty"`
}

// --- Billing ---

type CheckoutRequest struct {
	Plan     string `json:"plan"`
	Provider string `json:"provider"`
}

type CheckoutResponse struct {
	CheckoutURL string `json:"checkout_url"`
}

type SubscriptionResponse struct {
	Plan      string     `json:"plan"`
	Status    string     `json:"status"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Provider  string     `json:"provider"`
}

// --- Public ---

type PublicAuditRequest struct {
	IGUsername string `json:"ig_username"`
	Locale     string `json:"locale,omitempty"`
	MockScore  int    `json:"mock_score,omitempty"`
}

type PublicAuditResult struct {
	ID              string           `json:"id"`
	Username        string           `json:"username"`
	Score           int              `json:"score"`
	Strengths       []Strength       `json:"strengths"`
	Recommendations []Recommendation `json:"recommendations"`
	CreatedAt       string           `json:"created_at"`
}

type PublicAuditResponse struct {
	Status string       `json:"status"`
	Result *BrandDnaDTO `json:"result,omitempty"`
}

// --- Health ---

type HealthResponse struct {
	Status string `json:"status"`
	DB     string `json:"db"`
}
