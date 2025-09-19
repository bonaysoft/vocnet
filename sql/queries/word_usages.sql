-- name: CreateWordUsage :one
INSERT INTO word_usages (
  user_id, user_word_id, sentence_id, start_offset, end_offset, original_text, normalized_form, grammatical_role, usage_type, confidence, note
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
RETURNING *;

-- name: ListWordUsagesForWord :many
SELECT * FROM word_usages WHERE user_id = $1 AND user_word_id = $2 ORDER BY created_at DESC;

-- name: ListWordUsagesForSentence :many
SELECT * FROM word_usages WHERE sentence_id = $1 ORDER BY start_offset ASC;
