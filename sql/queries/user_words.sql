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
RETURNING user_words.*;

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
RETURNING user_words.*;

-- name: GetUserWord :one
SELECT user_words.*
FROM user_words
WHERE id = $1 AND user_id = $2;

-- name: FindUserWordByWord :one
SELECT user_words.*
FROM user_words
WHERE user_id = $1 AND word_normalized = lower($2)
LIMIT 1;

-- name: ListUserWords :many
SELECT
    sqlc.embed(user_words),
    sqlc.embed(words)
FROM user_words
LEFT JOIN words ON lower(user_words.word) = lower(words.text) AND user_words.language = words.language
WHERE user_id = sqlc.arg('user_id')
    AND (
        sqlc.arg('keyword')::text = ''
        OR word ILIKE '%' || sqlc.arg('keyword') || '%'
        OR notes ILIKE '%' || sqlc.arg('keyword') || '%'
    )
    AND (
        COALESCE(array_length(sqlc.arg('words')::text[], 1), 0) = 0
        OR word_normalized = ANY(sqlc.arg('words')::text[])
    )
ORDER BY
    CASE WHEN sqlc.arg(primary_key) = 'created_at' AND sqlc.arg(primary_desc) THEN user_words.created_at END DESC NULLS LAST,
    CASE WHEN sqlc.arg(primary_key) = 'created_at' AND NOT sqlc.arg(primary_desc) THEN user_words.created_at END ASC NULLS LAST,
    CASE WHEN sqlc.arg(primary_key) = 'updated_at' AND sqlc.arg(primary_desc) THEN user_words.updated_at END DESC NULLS LAST,
    CASE WHEN sqlc.arg(primary_key) = 'updated_at' AND NOT sqlc.arg(primary_desc) THEN user_words.updated_at END ASC NULLS LAST,
    CASE WHEN sqlc.arg(primary_key) = 'word' AND sqlc.arg(primary_desc) THEN word END DESC NULLS LAST,
    CASE WHEN sqlc.arg(primary_key) = 'word' AND NOT sqlc.arg(primary_desc) THEN word END ASC NULLS LAST,
    CASE WHEN sqlc.arg(primary_key) = 'mastery_overall' AND sqlc.arg(primary_desc) THEN mastery_overall END DESC NULLS LAST,
    CASE WHEN sqlc.arg(primary_key) = 'mastery_overall' AND NOT sqlc.arg(primary_desc) THEN mastery_overall END ASC NULLS LAST,
    CASE WHEN sqlc.arg(primary_key) = 'id' AND sqlc.arg(primary_desc) THEN user_words.id END DESC,
    CASE WHEN sqlc.arg(primary_key) = 'id' AND NOT sqlc.arg(primary_desc) THEN user_words.id END ASC,
    CASE WHEN sqlc.arg(secondary_key) = 'created_at' AND sqlc.arg(secondary_desc) THEN user_words.created_at END DESC NULLS LAST,
    CASE WHEN sqlc.arg(secondary_key) = 'created_at' AND NOT sqlc.arg(secondary_desc) THEN user_words.created_at END ASC NULLS LAST,
    CASE WHEN sqlc.arg(secondary_key) = 'updated_at' AND sqlc.arg(secondary_desc) THEN user_words.updated_at END DESC NULLS LAST,
    CASE WHEN sqlc.arg(secondary_key) = 'updated_at' AND NOT sqlc.arg(secondary_desc) THEN user_words.updated_at END ASC NULLS LAST,
    CASE WHEN sqlc.arg(secondary_key) = 'word' AND sqlc.arg(secondary_desc) THEN word END DESC NULLS LAST,
    CASE WHEN sqlc.arg(secondary_key) = 'word' AND NOT sqlc.arg(secondary_desc) THEN word END ASC NULLS LAST,
    CASE WHEN sqlc.arg(secondary_key) = 'mastery_overall' AND sqlc.arg(secondary_desc) THEN mastery_overall END DESC NULLS LAST,
    CASE WHEN sqlc.arg(secondary_key) = 'mastery_overall' AND NOT sqlc.arg(secondary_desc) THEN mastery_overall END ASC NULLS LAST,
    CASE WHEN sqlc.arg(secondary_key) = 'id' AND sqlc.arg(secondary_desc) THEN user_words.id END DESC,
    CASE WHEN sqlc.arg(secondary_key) = 'id' AND NOT sqlc.arg(secondary_desc) THEN user_words.id END ASC,
    user_words.id ASC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- name: CountUserWords :one
SELECT COUNT(*)
FROM user_words
WHERE user_id = sqlc.arg('user_id')
    AND (
        sqlc.arg('keyword')::text = ''
        OR word ILIKE '%' || sqlc.arg('keyword') || '%'
        OR notes ILIKE '%' || sqlc.arg('keyword') || '%'
    )
    AND (
        COALESCE(array_length(sqlc.arg('words')::text[], 1), 0) = 0
        OR word_normalized = ANY(sqlc.arg('words')::text[])
    );

-- name: DeleteUserWord :execresult
DELETE FROM user_words
WHERE id = $1 AND user_id = $2;
