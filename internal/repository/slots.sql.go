package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type CreateSlotParams struct {
	PlanID        uuid.UUID
	DayNumber     int32
	ScheduledDate time.Time
	ScheduledTime time.Time
	Title         string
	ContentType   string
	Format        string
	Brief         json.RawMessage
	Caption       string
	Hashtags      []string
	Cta           *string
}

func (q *Queries) CreateSlot(ctx context.Context, arg CreateSlotParams) (ContentSlot, error) {
	row := q.db.QueryRow(ctx,
		`INSERT INTO content_slots (plan_id, day_number, scheduled_date, scheduled_time,
		title, content_type, format, brief, caption, hashtags, cta)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING *`,
		arg.PlanID, arg.DayNumber, arg.ScheduledDate, arg.ScheduledTime,
		arg.Title, arg.ContentType, arg.Format, arg.Brief, arg.Caption, arg.Hashtags, arg.Cta,
	)
	return scanSlot(row)
}

func (q *Queries) GetSlotsByPlanID(ctx context.Context, planID uuid.UUID) ([]ContentSlot, error) {
	rows, err := q.db.Query(ctx,
		`SELECT * FROM content_slots WHERE plan_id = $1 ORDER BY day_number`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ContentSlot
	for rows.Next() {
		var s ContentSlot
		if err := rows.Scan(&s.ID, &s.PlanID, &s.DayNumber, &s.ScheduledDate, &s.ScheduledTime,
			&s.Title, &s.ContentType, &s.Format, &s.Brief, &s.Caption, &s.Hashtags, &s.Cta, &s.Media,
			&s.Status, &s.IsUserContent, &s.PublishedAt, &s.IgPostID, &s.IgPostUrl, &s.ErrorMessage,
			&s.RetryCount, &s.RegenerationCount, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, nil
}

func (q *Queries) GetSlotByID(ctx context.Context, id uuid.UUID) (ContentSlot, error) {
	row := q.db.QueryRow(ctx, `SELECT * FROM content_slots WHERE id = $1`, id)
	return scanSlot(row)
}

type UpdateSlotParams struct {
	ID            uuid.UUID
	Caption       *string
	Hashtags      []string
	ScheduledTime *time.Time
	ScheduledDate *time.Time
	Status        *string
	IsUserContent *bool
}

func (q *Queries) UpdateSlot(ctx context.Context, arg UpdateSlotParams) (ContentSlot, error) {
	row := q.db.QueryRow(ctx,
		`UPDATE content_slots SET
		caption = COALESCE($2, caption),
		hashtags = COALESCE($3, hashtags),
		scheduled_time = COALESCE($4, scheduled_time),
		scheduled_date = COALESCE($5, scheduled_date),
		status = COALESCE($6, status),
		is_user_content = COALESCE($7, is_user_content),
		updated_at = NOW()
		WHERE id = $1 RETURNING *`,
		arg.ID, arg.Caption, arg.Hashtags, arg.ScheduledTime, arg.ScheduledDate, arg.Status, arg.IsUserContent,
	)
	return scanSlot(row)
}

type UpdateSlotMediaParams struct {
	ID    uuid.UUID
	Media json.RawMessage
}

func (q *Queries) UpdateSlotMedia(ctx context.Context, arg UpdateSlotMediaParams) (ContentSlot, error) {
	row := q.db.QueryRow(ctx,
		`UPDATE content_slots SET media = $2, updated_at = NOW() WHERE id = $1 RETURNING *`,
		arg.ID, arg.Media,
	)
	return scanSlot(row)
}

type UpdateSlotStatusParams struct {
	ID           uuid.UUID
	Status       string
	ErrorMessage *string
}

func (q *Queries) UpdateSlotStatus(ctx context.Context, arg UpdateSlotStatusParams) error {
	_, err := q.db.Exec(ctx,
		`UPDATE content_slots SET status = $2, error_message = $3, updated_at = NOW() WHERE id = $1`,
		arg.ID, arg.Status, arg.ErrorMessage)
	return err
}

type UpdateSlotPublishedParams struct {
	ID        uuid.UUID
	IgPostID  *string
	IgPostUrl *string
}

func (q *Queries) UpdateSlotPublished(ctx context.Context, arg UpdateSlotPublishedParams) error {
	_, err := q.db.Exec(ctx,
		`UPDATE content_slots SET status = 'published', published_at = NOW(),
		ig_post_id = $2, ig_post_url = $3, updated_at = NOW() WHERE id = $1`,
		arg.ID, arg.IgPostID, arg.IgPostUrl)
	return err
}

func (q *Queries) IncrementSlotRetry(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.Exec(ctx,
		`UPDATE content_slots SET retry_count = retry_count + 1, updated_at = NOW() WHERE id = $1`, id)
	return err
}

func (q *Queries) IncrementSlotRegeneration(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.Exec(ctx,
		`UPDATE content_slots SET regeneration_count = regeneration_count + 1, updated_at = NOW() WHERE id = $1`, id)
	return err
}

func (q *Queries) ApproveAllDraftSlots(ctx context.Context, planID uuid.UUID) (int64, error) {
	tag, err := q.db.Exec(ctx,
		`UPDATE content_slots SET status = 'approved', updated_at = NOW()
		WHERE plan_id = $1 AND status = 'draft'`, planID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (q *Queries) CountApprovedWithoutMedia(ctx context.Context, planID uuid.UUID) (int32, error) {
	var count int32
	err := q.db.QueryRow(ctx,
		`SELECT COUNT(*)::int FROM content_slots WHERE plan_id = $1 AND status = 'approved' AND media::text = '[]'`,
		planID).Scan(&count)
	return count, err
}

func (q *Queries) QueueApprovedSlots(ctx context.Context, planID uuid.UUID) (int64, error) {
	tag, err := q.db.Exec(ctx,
		`UPDATE content_slots SET status = 'queued', updated_at = NOW()
		WHERE plan_id = $1 AND status = 'approved' AND media::text != '[]'`, planID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

type SlotWithAccount struct {
	ContentSlot
	InstagramAccountID uuid.UUID
}

func (q *Queries) GetSlotsReadyToPublish(ctx context.Context) ([]SlotWithAccount, error) {
	rows, err := q.db.Query(ctx,
		`SELECT cs.*, cp.instagram_account_id
		FROM content_slots cs
		JOIN content_plans cp ON cs.plan_id = cp.id
		WHERE cs.status = 'approved'
		  AND cs.scheduled_date = CURRENT_DATE
		  AND cs.scheduled_time <= CURRENT_TIME
		  AND cs.media::text != '[]'
		ORDER BY cs.scheduled_time`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []SlotWithAccount
	for rows.Next() {
		var s SlotWithAccount
		if err := rows.Scan(&s.ID, &s.PlanID, &s.DayNumber, &s.ScheduledDate, &s.ScheduledTime,
			&s.Title, &s.ContentType, &s.Format, &s.Brief, &s.Caption, &s.Hashtags, &s.Cta, &s.Media,
			&s.Status, &s.IsUserContent, &s.PublishedAt, &s.IgPostID, &s.IgPostUrl, &s.ErrorMessage,
			&s.RetryCount, &s.RegenerationCount, &s.CreatedAt, &s.UpdatedAt,
			&s.InstagramAccountID); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, nil
}

func (q *Queries) SkipSlotsMissingMedia(ctx context.Context) (int64, error) {
	tag, err := q.db.Exec(ctx,
		`UPDATE content_slots SET
		  status = 'skipped',
		  error_message = 'No media uploaded',
		  updated_at = NOW()
		WHERE status = 'approved'
		  AND scheduled_date = CURRENT_DATE
		  AND scheduled_time <= CURRENT_TIME
		  AND media::text = '[]'`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

type slotScanner interface {
	Scan(dest ...interface{}) error
}

func scanSlot(row slotScanner) (ContentSlot, error) {
	var s ContentSlot
	err := row.Scan(&s.ID, &s.PlanID, &s.DayNumber, &s.ScheduledDate, &s.ScheduledTime,
		&s.Title, &s.ContentType, &s.Format, &s.Brief, &s.Caption, &s.Hashtags, &s.Cta, &s.Media,
		&s.Status, &s.IsUserContent, &s.PublishedAt, &s.IgPostID, &s.IgPostUrl, &s.ErrorMessage,
		&s.RetryCount, &s.RegenerationCount, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}
