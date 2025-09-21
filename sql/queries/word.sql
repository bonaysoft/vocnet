-- name: CreateWord :one
INSERT INTO words (
    lemma, language, phonetic, pos, definition, translation, exchange, tags
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
RETURNING id, lemma, language, phonetic, pos, definition, translation, exchange, tags, created_at;

-- name: GetWord :one
SELECT id, lemma, language, phonetic, pos, definition, translation, exchange, tags, created_at
FROM words
WHERE id = $1 LIMIT 1;

-- name: ListWords :many
SELECT id, lemma, language, phonetic, pos, definition, translation, exchange, tags, created_at
FROM words
WHERE ($1::text IS NULL OR language = $1)
  AND ($2::text IS NULL OR lemma ILIKE $2 || '%')
ORDER BY lemma ASC
LIMIT $3 OFFSET $4;

-- name: LookupWord :one
SELECT id, lemma, language, phonetic, pos, definition, translation, exchange, tags, created_at
FROM words
WHERE lower(lemma) = lower($1) AND language = $2
LIMIT 1;
