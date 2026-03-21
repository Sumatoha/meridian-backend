package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UpsertScrapedPostParams struct {
	InstagramAccountID uuid.UUID
	IgPostID           string
	PostType           *string
	Caption            *string
	Hashtags           []string
	LikesCount         *int32
	CommentsCount      *int32
	PostedAt           *time.Time
	ThumbnailUrl       *string
}

func (q *Queries) UpsertScrapedPost(ctx context.Context, arg UpsertScrapedPostParams) error {
	_, err := q.db.Exec(ctx,
		`INSERT INTO scraped_posts (instagram_account_id, ig_post_id, post_type,
		caption, hashtags, likes_count, comments_count, posted_at, thumbnail_url)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (instagram_account_id, ig_post_id) DO UPDATE SET
		caption = EXCLUDED.caption, hashtags = EXCLUDED.hashtags,
		likes_count = EXCLUDED.likes_count, comments_count = EXCLUDED.comments_count, scraped_at = NOW()`,
		arg.InstagramAccountID, arg.IgPostID, arg.PostType,
		arg.Caption, arg.Hashtags, arg.LikesCount, arg.CommentsCount,
		arg.PostedAt, arg.ThumbnailUrl,
	)
	return err
}
