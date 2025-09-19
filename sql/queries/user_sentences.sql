-- name: LinkUserSentence :one
INSERT INTO user_sentences (
  user_id, sentence_id, is_starred, familiarity, private_note
) VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (user_id, sentence_id) DO UPDATE SET updated_at = NOW()
RETURNING *;

-- name: UpdateUserSentenceReview :one
UPDATE user_sentences
SET last_review_at = NOW(), next_review_at = $2, review_interval_days = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateUserSentenceFamiliarity :one
UPDATE user_sentences
SET familiarity = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;
