package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

type CreateBrandDnaParams struct {
	InstagramAccountID  uuid.UUID
	Score               int32
	Tone                string
	VisualStyle         *string
	StrongTopics        []string
	WeakAreas           []string
	BestFormats         []string
	BestPostingTimes    []string
	AvgPostingFrequency *string
	HashtagStrategy     *string
	Strengths           json.RawMessage
	Recommendations     json.RawMessage
	RawAnalysis         json.RawMessage
}

func (q *Queries) CreateBrandDna(ctx context.Context, arg CreateBrandDnaParams) (BrandDna, error) {
	row := q.db.QueryRow(ctx,
		`INSERT INTO brand_dna (instagram_account_id, score, tone, visual_style,
		strong_topics, weak_areas, best_formats, best_posting_times,
		avg_posting_frequency, hashtag_strategy, strengths, recommendations, raw_analysis)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING *`,
		arg.InstagramAccountID, arg.Score, arg.Tone, arg.VisualStyle,
		arg.StrongTopics, arg.WeakAreas, arg.BestFormats, arg.BestPostingTimes,
		arg.AvgPostingFrequency, arg.HashtagStrategy, arg.Strengths, arg.Recommendations, arg.RawAnalysis,
	)
	var d BrandDna
	err := row.Scan(&d.ID, &d.InstagramAccountID, &d.Score, &d.Tone, &d.VisualStyle,
		&d.StrongTopics, &d.WeakAreas, &d.BestFormats, &d.BestPostingTimes,
		&d.AvgPostingFrequency, &d.HashtagStrategy, &d.Strengths, &d.Recommendations, &d.RawAnalysis, &d.CreatedAt)
	return d, err
}

func (q *Queries) GetLatestBrandDna(ctx context.Context, instagramAccountID uuid.UUID) (BrandDna, error) {
	row := q.db.QueryRow(ctx,
		`SELECT * FROM brand_dna WHERE instagram_account_id = $1 ORDER BY created_at DESC LIMIT 1`,
		instagramAccountID)
	var d BrandDna
	err := row.Scan(&d.ID, &d.InstagramAccountID, &d.Score, &d.Tone, &d.VisualStyle,
		&d.StrongTopics, &d.WeakAreas, &d.BestFormats, &d.BestPostingTimes,
		&d.AvgPostingFrequency, &d.HashtagStrategy, &d.Strengths, &d.Recommendations, &d.RawAnalysis, &d.CreatedAt)
	return d, err
}
