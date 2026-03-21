package repository

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID             uuid.UUID  `json:"id"`
	SupabaseUserID uuid.UUID  `json:"supabase_user_id"`
	Email          string     `json:"email"`
	Plan           string     `json:"plan"`
	PlanExpiresAt  *time.Time `json:"plan_expires_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type InstagramAccount struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	IgUsername       string     `json:"ig_username"`
	IgUserID         *string    `json:"ig_user_id"`
	AccessToken      *string    `json:"access_token"`
	TokenExpiresAt   *time.Time `json:"token_expires_at"`
	ProfilePicUrl    *string    `json:"profile_pic_url"`
	FollowersCount   *int32     `json:"followers_count"`
	IsOauthConnected bool       `json:"is_oauth_connected"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type BrandSetting struct {
	ID                    uuid.UUID        `json:"id"`
	InstagramAccountID    uuid.UUID        `json:"instagram_account_id"`
	ContentGoal           string           `json:"content_goal"`
	ToneTraits            []string         `json:"tone_traits"`
	ToneCustomNote        *string          `json:"tone_custom_note"`
	MixUseful             int32            `json:"mix_useful"`
	MixSelling            int32            `json:"mix_selling"`
	MixPersonal           int32            `json:"mix_personal"`
	MixEntertaining       int32            `json:"mix_entertaining"`
	FormatReelsEnabled    bool             `json:"format_reels_enabled"`
	FormatReelsPct        int32            `json:"format_reels_pct"`
	FormatCarouselEnabled bool             `json:"format_carousel_enabled"`
	FormatCarouselPct     int32            `json:"format_carousel_pct"`
	FormatPhotoEnabled    bool             `json:"format_photo_enabled"`
	FormatPhotoPct        int32            `json:"format_photo_pct"`
	BannedTopics          []string         `json:"banned_topics"`
	BannedWords           []string         `json:"banned_words"`
	CompetitorNames       []string         `json:"competitor_names"`
	ContentRestrictions   []string         `json:"content_restrictions"`
	CustomRules           *string          `json:"custom_rules"`
	ProductsServices      *string          `json:"products_services"`
	TargetAudience        *string          `json:"target_audience"`
	Usp                   *string          `json:"usp"`
	TeamMembers           json.RawMessage  `json:"team_members"`
	LocationAddress       *string          `json:"location_address"`
	WorkingHours          *string          `json:"working_hours"`
	UpcomingEvents        json.RawMessage  `json:"upcoming_events"`
	ContentLanguage       string           `json:"content_language"`
	PostingFrequency      string           `json:"posting_frequency"`
	Niche                 *string          `json:"niche"`
	CreatedAt             time.Time        `json:"created_at"`
	UpdatedAt             time.Time        `json:"updated_at"`
}

type BrandDna struct {
	ID                  uuid.UUID       `json:"id"`
	InstagramAccountID  uuid.UUID       `json:"instagram_account_id"`
	Score               int32           `json:"score"`
	Tone                string          `json:"tone"`
	VisualStyle         *string         `json:"visual_style"`
	StrongTopics        []string        `json:"strong_topics"`
	WeakAreas           []string        `json:"weak_areas"`
	BestFormats         []string        `json:"best_formats"`
	BestPostingTimes    []string        `json:"best_posting_times"`
	AvgPostingFrequency *string         `json:"avg_posting_frequency"`
	HashtagStrategy     *string         `json:"hashtag_strategy"`
	Strengths           json.RawMessage `json:"strengths"`
	Recommendations     json.RawMessage `json:"recommendations"`
	RawAnalysis         json.RawMessage `json:"raw_analysis"`
	CreatedAt           time.Time       `json:"created_at"`
}

type ContentPlan struct {
	ID                 uuid.UUID `json:"id"`
	InstagramAccountID uuid.UUID `json:"instagram_account_id"`
	Title              string    `json:"title"`
	StartDate          time.Time `json:"start_date"`
	EndDate            time.Time `json:"end_date"`
	Status             string    `json:"status"`
	TotalSlots         int32     `json:"total_slots"`
	ApprovedSlots      int32     `json:"approved_slots"`
	PublishedSlots     int32     `json:"published_slots"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type ContentSlot struct {
	ID                uuid.UUID       `json:"id"`
	PlanID            uuid.UUID       `json:"plan_id"`
	DayNumber         int32           `json:"day_number"`
	ScheduledDate     time.Time       `json:"scheduled_date"`
	ScheduledTime     time.Time       `json:"scheduled_time"`
	Title             string          `json:"title"`
	ContentType       string          `json:"content_type"`
	Format            string          `json:"format"`
	Brief             json.RawMessage `json:"brief"`
	Caption           string          `json:"caption"`
	Hashtags          []string        `json:"hashtags"`
	Cta               *string         `json:"cta"`
	Media             json.RawMessage `json:"media"`
	Status            string          `json:"status"`
	IsUserContent     bool            `json:"is_user_content"`
	PublishedAt       *time.Time      `json:"published_at"`
	IgPostID          *string         `json:"ig_post_id"`
	IgPostUrl         *string         `json:"ig_post_url"`
	ErrorMessage      *string         `json:"error_message"`
	RetryCount        int32           `json:"retry_count"`
	RegenerationCount int32           `json:"regeneration_count"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type ScrapedPost struct {
	ID                 uuid.UUID  `json:"id"`
	InstagramAccountID uuid.UUID  `json:"instagram_account_id"`
	IgPostID           string     `json:"ig_post_id"`
	PostType           *string    `json:"post_type"`
	Caption            *string    `json:"caption"`
	Hashtags           []string   `json:"hashtags"`
	LikesCount         *int32     `json:"likes_count"`
	CommentsCount      *int32     `json:"comments_count"`
	PostedAt           *time.Time `json:"posted_at"`
	ThumbnailUrl       *string    `json:"thumbnail_url"`
	ScrapedAt          time.Time  `json:"scraped_at"`
}

type Payment struct {
	ID                 uuid.UUID  `json:"id"`
	UserID             uuid.UUID  `json:"user_id"`
	Provider           string     `json:"provider"`
	ExternalID         string     `json:"external_id"`
	AmountCents        int32      `json:"amount_cents"`
	Currency           string     `json:"currency"`
	Status             string     `json:"status"`
	Plan               string     `json:"plan"`
	CurrentPeriodStart *time.Time `json:"current_period_start"`
	CurrentPeriodEnd   *time.Time `json:"current_period_end"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}
