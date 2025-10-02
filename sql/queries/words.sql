-- Single-table JSONB queries for words

-- name: CreateWord :one
INSERT INTO words (text, language, word_type, lemma, phonetics, meanings, tags)
VALUES ($1,$2,$3,$4,$5,$6,$7)
RETURNING id, text, language, word_type, lemma, phonetics, meanings, tags, created_at;

-- Get specific word by composite key
-- name: GetWord :one
SELECT id, text, language, word_type, lemma, phonetics, meanings, tags, created_at
FROM words
WHERE language = $1 AND text = $2 AND word_type = $3
LIMIT 1;

-- name: GetWordByID :one
SELECT id, text, language, word_type, lemma, phonetics, meanings, tags, created_at
FROM words
WHERE id = $1
LIMIT 1;

-- name: UpdateWord :one
UPDATE words
SET text = $2,
    language = $3,
    word_type = $4,
    lemma = $5,
  phonetics = $6,
    meanings = $7,
    tags = $8
WHERE id = $1
RETURNING id, text, language, word_type, lemma, phonetics, meanings, tags, created_at;

-- name: DeleteWord :execrows
DELETE FROM words
WHERE id = $1;

-- name: CountWords :one
SELECT COUNT(*)
FROM words
WHERE (sqlc.arg(language_filter) = '' OR language = sqlc.arg(language_filter))
  AND (sqlc.arg(keyword_filter) = '' OR text ILIKE sqlc.arg(keyword_filter) || '%')
  AND (sqlc.arg(word_type_filter) = '' OR word_type = sqlc.arg(word_type_filter));

-- Lookup any word entry by text (prefer lemma). We return the lemma row if exists, else any row.
-- name: LookupWord :one
WITH candidates AS (
    SELECT *, CASE WHEN word_type='lemma' THEN 0 ELSE 1 END AS priority
    FROM words
    WHERE lower(text) = lower($1) AND language = $2
)
SELECT id, text, language, word_type, lemma, phonetics, meanings, tags, created_at
FROM candidates
ORDER BY priority ASC
LIMIT 1;

-- name: ListWords :many
SELECT id, text, language, word_type, lemma, phonetics, meanings, tags, created_at
FROM words
WHERE (sqlc.arg(language_filter) = '' OR language = sqlc.arg(language_filter))
  AND (sqlc.arg(keyword_filter) = '' OR text ILIKE sqlc.arg(keyword_filter) || '%')
  AND (sqlc.arg(word_type_filter) = '' OR word_type = sqlc.arg(word_type_filter))
ORDER BY text ASC
LIMIT sqlc.arg(result_limit) OFFSET sqlc.arg(result_offset);

-- name: ListWordsByPOS :many
SELECT id, text, language, word_type, lemma, phonetics, meanings, tags, created_at
FROM words
WHERE meanings @> $1::jsonb
ORDER BY text ASC
LIMIT $2 OFFSET $3;

-- List all variants/inflections for a lemma
-- name: ListInflections :many
SELECT id, text, language, word_type, lemma, phonetics, meanings, tags, created_at
FROM words
WHERE language = $1 AND lemma = $2
ORDER BY text ASC;
