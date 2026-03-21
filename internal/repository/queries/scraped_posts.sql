-- name: UpsertScrapedPost :exec
INSERT INTO scraped_posts (
    instagram_account_id, ig_post_id, post_type,
    caption, hashtags, likes_count, comments_count,
    posted_at, thumbnail_url
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (instagram_account_id, ig_post_id) DO UPDATE SET
    caption = EXCLUDED.caption,
    hashtags = EXCLUDED.hashtags,
    likes_count = EXCLUDED.likes_count,
    comments_count = EXCLUDED.comments_count,
    scraped_at = NOW();

-- name: GetScrapedPosts :many
SELECT * FROM scraped_posts
WHERE instagram_account_id = $1
ORDER BY posted_at DESC
LIMIT $2;
