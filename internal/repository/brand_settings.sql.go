package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

func (q *Queries) GetBrandSettings(ctx context.Context, instagramAccountID uuid.UUID) (BrandSetting, error) {
	row := q.db.QueryRow(ctx, `SELECT * FROM brand_settings WHERE instagram_account_id = $1`, instagramAccountID)
	var s BrandSetting
	err := row.Scan(
		&s.ID, &s.InstagramAccountID,
		&s.ContentGoal, &s.ToneTraits, &s.ToneCustomNote,
		&s.MixUseful, &s.MixSelling, &s.MixPersonal, &s.MixEntertaining,
		&s.FormatReelsEnabled, &s.FormatReelsPct,
		&s.FormatCarouselEnabled, &s.FormatCarouselPct,
		&s.FormatPhotoEnabled, &s.FormatPhotoPct,
		&s.BannedTopics, &s.BannedWords, &s.CompetitorNames, &s.ContentRestrictions, &s.CustomRules,
		&s.ProductsServices, &s.TargetAudience, &s.Usp, &s.TeamMembers,
		&s.LocationAddress, &s.WorkingHours, &s.UpcomingEvents,
		&s.ContentLanguage, &s.PostingFrequency, &s.Niche,
		&s.CreatedAt, &s.UpdatedAt,
	)
	return s, err
}

type UpsertBrandSettingsParams struct {
	InstagramAccountID  uuid.UUID
	ContentGoal         string
	ToneTraits          []string
	ToneCustomNote      *string
	MixUseful           int32
	MixSelling          int32
	MixPersonal         int32
	MixEntertaining     int32
	FormatReelsEnabled  bool
	FormatReelsPct      int32
	FormatCarouselEnabled bool
	FormatCarouselPct   int32
	FormatPhotoEnabled  bool
	FormatPhotoPct      int32
	BannedTopics        []string
	BannedWords         []string
	CompetitorNames     []string
	ContentRestrictions []string
	CustomRules         *string
	ProductsServices    *string
	TargetAudience      *string
	Usp                 *string
	TeamMembers         json.RawMessage
	LocationAddress     *string
	WorkingHours        *string
	UpcomingEvents      json.RawMessage
	ContentLanguage     string
	PostingFrequency    string
	Niche               *string
}

func (q *Queries) UpsertBrandSettings(ctx context.Context, arg UpsertBrandSettingsParams) (BrandSetting, error) {
	row := q.db.QueryRow(ctx,
		`INSERT INTO brand_settings (
			instagram_account_id, content_goal, tone_traits, tone_custom_note,
			mix_useful, mix_selling, mix_personal, mix_entertaining,
			format_reels_enabled, format_reels_pct, format_carousel_enabled, format_carousel_pct,
			format_photo_enabled, format_photo_pct,
			banned_topics, banned_words, competitor_names, content_restrictions, custom_rules,
			products_services, target_audience, usp, team_members,
			location_address, working_hours, upcoming_events,
			content_language, posting_frequency, niche
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29)
		ON CONFLICT (instagram_account_id) DO UPDATE SET
			content_goal=EXCLUDED.content_goal, tone_traits=EXCLUDED.tone_traits, tone_custom_note=EXCLUDED.tone_custom_note,
			mix_useful=EXCLUDED.mix_useful, mix_selling=EXCLUDED.mix_selling, mix_personal=EXCLUDED.mix_personal, mix_entertaining=EXCLUDED.mix_entertaining,
			format_reels_enabled=EXCLUDED.format_reels_enabled, format_reels_pct=EXCLUDED.format_reels_pct,
			format_carousel_enabled=EXCLUDED.format_carousel_enabled, format_carousel_pct=EXCLUDED.format_carousel_pct,
			format_photo_enabled=EXCLUDED.format_photo_enabled, format_photo_pct=EXCLUDED.format_photo_pct,
			banned_topics=EXCLUDED.banned_topics, banned_words=EXCLUDED.banned_words,
			competitor_names=EXCLUDED.competitor_names, content_restrictions=EXCLUDED.content_restrictions, custom_rules=EXCLUDED.custom_rules,
			products_services=EXCLUDED.products_services, target_audience=EXCLUDED.target_audience, usp=EXCLUDED.usp, team_members=EXCLUDED.team_members,
			location_address=EXCLUDED.location_address, working_hours=EXCLUDED.working_hours, upcoming_events=EXCLUDED.upcoming_events,
			content_language=EXCLUDED.content_language, posting_frequency=EXCLUDED.posting_frequency, niche=EXCLUDED.niche, updated_at=NOW()
		RETURNING *`,
		arg.InstagramAccountID, arg.ContentGoal, arg.ToneTraits, arg.ToneCustomNote,
		arg.MixUseful, arg.MixSelling, arg.MixPersonal, arg.MixEntertaining,
		arg.FormatReelsEnabled, arg.FormatReelsPct, arg.FormatCarouselEnabled, arg.FormatCarouselPct,
		arg.FormatPhotoEnabled, arg.FormatPhotoPct,
		arg.BannedTopics, arg.BannedWords, arg.CompetitorNames, arg.ContentRestrictions, arg.CustomRules,
		arg.ProductsServices, arg.TargetAudience, arg.Usp, arg.TeamMembers,
		arg.LocationAddress, arg.WorkingHours, arg.UpcomingEvents,
		arg.ContentLanguage, arg.PostingFrequency, arg.Niche,
	)
	var s BrandSetting
	err := row.Scan(
		&s.ID, &s.InstagramAccountID,
		&s.ContentGoal, &s.ToneTraits, &s.ToneCustomNote,
		&s.MixUseful, &s.MixSelling, &s.MixPersonal, &s.MixEntertaining,
		&s.FormatReelsEnabled, &s.FormatReelsPct,
		&s.FormatCarouselEnabled, &s.FormatCarouselPct,
		&s.FormatPhotoEnabled, &s.FormatPhotoPct,
		&s.BannedTopics, &s.BannedWords, &s.CompetitorNames, &s.ContentRestrictions, &s.CustomRules,
		&s.ProductsServices, &s.TargetAudience, &s.Usp, &s.TeamMembers,
		&s.LocationAddress, &s.WorkingHours, &s.UpcomingEvents,
		&s.ContentLanguage, &s.PostingFrequency, &s.Niche,
		&s.CreatedAt, &s.UpdatedAt,
	)
	return s, err
}
