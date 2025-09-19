-- name: CreateWordRelation :one
INSERT INTO word_relations (
  user_id, word_a_id, word_b_id, relation_type, subtype, is_bidirectional, weight, note, created_source, metadata
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
RETURNING *;

-- name: ListWordRelationsForWord :many
SELECT * FROM word_relations
WHERE user_id = $1 AND (word_a_id = $2 OR word_b_id = $2) AND is_active = TRUE;

-- name: UpdateWordRelationWeight :one
UPDATE word_relations
SET weight = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivateWordRelation :exec
UPDATE word_relations SET is_active = FALSE, updated_at = NOW() WHERE id = $1;
