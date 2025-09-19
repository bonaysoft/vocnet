-- name: CreateUserWord :one
INSERT INTO user_words (
  user_id, word_id, custom_text
) VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserWord :one
SELECT * FROM user_words WHERE id = $1;

-- name: ListUserWords :many
SELECT * FROM user_words
WHERE user_id = $1 AND is_active = TRUE
ORDER BY next_review_at NULLS FIRST, mastery_overall DESC
LIMIT $2 OFFSET $3;

-- name: UpdateUserWordMastery :one
UPDATE user_words
SET mastery_listen = $2,
    mastery_read = $3,
    mastery_spell = $4,
    mastery_pronounce = $5,
    mastery_use = $6,
    mastery_overall = $7,
    status = $8,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ScheduleUserWordReview :exec
UPDATE user_words
SET next_review_at = $2,
    review_interval_days = $3,
    last_review_at = NOW(),
    updated_at = NOW()
WHERE id = $1;

-- name: SoftDeleteUserWord :exec
UPDATE user_words SET is_active = FALSE, updated_at = NOW() WHERE id = $1;
