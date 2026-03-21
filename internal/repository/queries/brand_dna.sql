-- name: CreateBrandDna :one
INSERT INTO brand_dna (
    instagram_account_id, score, tone, visual_style,
    strong_topics, weak_areas, best_formats, best_posting_times,
    avg_posting_frequency, hashtag_strategy,
    strengths, recommendations, raw_analysis
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: GetLatestBrandDna :one
SELECT * FROM brand_dna
WHERE instagram_account_id = $1
ORDER BY created_at DESC
LIMIT 1;
