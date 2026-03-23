-- name: CreateSlot :one
INSERT INTO content_slots (
    plan_id, day_number, scheduled_date, scheduled_time,
    title, content_type, format, brief, caption, hashtags, cta
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetSlotsByPlanID :many
SELECT * FROM content_slots WHERE plan_id = $1 ORDER BY day_number;

-- name: GetSlotByID :one
SELECT * FROM content_slots WHERE id = $1;

-- name: UpdateSlot :one
UPDATE content_slots SET
    caption = COALESCE(sqlc.narg('caption'), caption),
    hashtags = COALESCE(sqlc.narg('hashtags'), hashtags),
    scheduled_time = COALESCE(sqlc.narg('scheduled_time'), scheduled_time),
    scheduled_date = COALESCE(sqlc.narg('scheduled_date'), scheduled_date),
    status = COALESCE(sqlc.narg('status'), status),
    is_user_content = COALESCE(sqlc.narg('is_user_content'), is_user_content),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateSlotMedia :one
UPDATE content_slots SET media = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateSlotStatus :exec
UPDATE content_slots SET
    status = $2,
    error_message = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateSlotPublished :exec
UPDATE content_slots SET
    status = 'published',
    published_at = NOW(),
    ig_post_id = $2,
    ig_post_url = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: IncrementSlotRetry :exec
UPDATE content_slots SET
    retry_count = retry_count + 1,
    updated_at = NOW()
WHERE id = $1;

-- name: IncrementSlotRegeneration :exec
UPDATE content_slots SET
    regeneration_count = regeneration_count + 1,
    updated_at = NOW()
WHERE id = $1;

-- name: ApproveAllDraftSlots :execrows
UPDATE content_slots SET status = 'approved', updated_at = NOW()
WHERE plan_id = $1 AND status = 'draft';

-- name: CountApprovedWithoutMedia :one
SELECT COUNT(*)::int FROM content_slots
WHERE plan_id = $1 AND status = 'approved' AND media::text = '[]';

-- name: QueueApprovedSlots :execrows
UPDATE content_slots SET status = 'queued', updated_at = NOW()
WHERE plan_id = $1 AND status = 'approved' AND media::text != '[]';

-- name: GetSlotsReadyToPublish :many
SELECT cs.*, cp.instagram_account_id
FROM content_slots cs
JOIN content_plans cp ON cs.plan_id = cp.id
WHERE cs.status = 'approved'
  AND cs.scheduled_date = CURRENT_DATE
  AND cs.scheduled_time <= CURRENT_TIME
  AND cs.media::text != '[]'
ORDER BY cs.scheduled_time;

-- name: SkipSlotsMissingMedia :execrows
UPDATE content_slots SET
  status = 'skipped',
  error_message = 'No media uploaded',
  updated_at = NOW()
WHERE status = 'approved'
  AND scheduled_date = CURRENT_DATE
  AND scheduled_time <= CURRENT_TIME
  AND media::text = '[]';

-- name: CountSlotsByPlanID :one
SELECT COUNT(*)::int FROM content_slots WHERE plan_id = $1;
