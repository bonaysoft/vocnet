-- name: CreateSentence :one
INSERT INTO sentences (
  content, content_norm, language, hash, length, token_count, created_source, metadata
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
RETURNING *;

-- name: GetSentenceByHash :one
SELECT * FROM sentences WHERE hash = $1;

-- name: GetSentence :one
SELECT * FROM sentences WHERE id = $1;

-- name: ListUserSentences :many
SELECT s.* FROM sentences s
JOIN user_sentences us ON us.sentence_id = s.id
WHERE us.user_id = $1
ORDER BY us.created_at DESC
LIMIT $2 OFFSET $3;
