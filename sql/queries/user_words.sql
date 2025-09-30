-- User word persistence queries

-- name: CreateUserWord :one
INSERT INTO user_words (
    user_id,
    word,
    language,
    mastery_listen,
    mastery_read,
    mastery_spell,
    mastery_pronounce,
    mastery_use,
    mastery_overall,
    review_last_review_at,
    review_next_review_at,
    review_interval_days,
    review_fail_count,
    query_count,
    notes,
    sentences,
    relations,
    created_by,
    created_at,
    updated_at
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11,
    $12,
    $13,
    $14,
    $15,
    $16,
    $17,
    $18,
    $19,
    $20
)
RETURNING
    id,
    user_id,
    word,
    language,
    mastery_listen,
    mastery_read,
    mastery_spell,
    mastery_pronounce,
    mastery_use,
    mastery_overall,
    review_last_review_at,
    review_next_review_at,
    review_interval_days,
    review_fail_count,
    query_count,
    notes,
    sentences,
    relations,
    created_by,
    created_at,
    updated_at;

-- name: UpdateUserWord :one
UPDATE user_words
SET
    word = $3,
    language = $4,
    mastery_listen = $5,
    mastery_read = $6,
    mastery_spell = $7,
    mastery_pronounce = $8,
    mastery_use = $9,
    mastery_overall = $10,
    review_last_review_at = $11,
    review_next_review_at = $12,
    review_interval_days = $13,
    review_fail_count = $14,
    query_count = $15,
    notes = $16,
    sentences = $17,
    relations = $18,
    created_by = $19,
    updated_at = $20
WHERE id = $1 AND user_id = $2
RETURNING
    id,
    user_id,
    word,
    language,
    mastery_listen,
    mastery_read,
    mastery_spell,
    mastery_pronounce,
    mastery_use,
    mastery_overall,
    review_last_review_at,
    review_next_review_at,
    review_interval_days,
    review_fail_count,
    query_count,
    notes,
    sentences,
    relations,
    created_by,
    created_at,
    updated_at;

-- name: GetUserWord :one
SELECT
    id,
    user_id,
    word,
    language,
    mastery_listen,
    mastery_read,
    mastery_spell,
    mastery_pronounce,
    mastery_use,
    mastery_overall,
    review_last_review_at,
    review_next_review_at,
    review_interval_days,
    review_fail_count,
    query_count,
    notes,
    sentences,
    relations,
    created_by,
    created_at,
    updated_at
FROM user_words
WHERE id = $1 AND user_id = $2;

-- name: FindUserWordByWord :one
SELECT
    id,
    user_id,
    word,
    language,
    mastery_listen,
    mastery_read,
    mastery_spell,
    mastery_pronounce,
    mastery_use,
    mastery_overall,
    review_last_review_at,
    review_next_review_at,
    review_interval_days,
    review_fail_count,
    query_count,
    notes,
    sentences,
    relations,
    created_by,
    created_at,
    updated_at
FROM user_words
WHERE user_id = $1 AND word_normalized = lower($2)
LIMIT 1;

-- name: ListUserWords :many
SELECT
    id,
    user_id,
    word,
    language,
    mastery_listen,
    mastery_read,
    mastery_spell,
    mastery_pronounce,
    mastery_use,
    mastery_overall,
    review_last_review_at,
    review_next_review_at,
    review_interval_days,
    review_fail_count,
    query_count,
    notes,
    sentences,
    relations,
    created_by,
    created_at,
    updated_at,
    COUNT(*) OVER() AS total_count
FROM user_words
WHERE user_id = $1
  AND (
        $2::text = ''
        OR word ILIKE '%' || $2 || '%'
        OR notes ILIKE '%' || $2 || '%'
      )
ORDER BY created_at DESC, id DESC
LIMIT $3
OFFSET $4;

-- name: DeleteUserWord :execresult
DELETE FROM user_words
WHERE id = $1 AND user_id = $2;
