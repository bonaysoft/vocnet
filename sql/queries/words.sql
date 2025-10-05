-- Single-table JSONB queries for words

-- name: CreateWord :one
INSERT INTO words (text, language, word_type, lemma, phonetics, meanings, tags, phrases, sentences, relations)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
RETURNING id, text, language, word_type, lemma, phonetics, meanings, tags, phrases, sentences, relations, created_at, updated_at;

-- Get specific word by composite key
-- name: GetWord :one
SELECT id, text, language, word_type, lemma, phonetics, meanings, tags, phrases, sentences, relations, created_at, updated_at
FROM words
WHERE language = $1 AND text = $2 AND word_type = $3
LIMIT 1;

-- name: GetWordByID :one
SELECT id, text, language, word_type, lemma, phonetics, meanings, tags, phrases, sentences, relations, created_at, updated_at
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
    tags = $8,
    phrases = $9,
    sentences = $10,
    relations = $11,
    updated_at = NOW()
WHERE id = $1
RETURNING id, text, language, word_type, lemma, phonetics, meanings, tags, phrases, sentences, relations, created_at, updated_at;

-- name: DeleteWord :execrows
DELETE FROM words
WHERE id = $1;

-- name: CountWords :one
SELECT COUNT(*)
FROM words
WHERE (sqlc.arg(language_filter) = '' OR language = sqlc.arg(language_filter))
  AND (sqlc.arg(keyword_filter) = '' OR text ILIKE sqlc.arg(keyword_filter) || '%')
  AND (sqlc.arg(word_type_filter) = '' OR word_type = sqlc.arg(word_type_filter))
  AND (
        COALESCE(array_length(sqlc.arg(words_filter)::text[], 1), 0) = 0
        OR lower(text) = ANY(sqlc.arg(words_filter)::text[])
      );

-- Lookup any word entry by text (prefer lemma). We return the lemma row if exists, else any row.
-- name: LookupWord :one
SELECT id, text, language, word_type, lemma, phonetics, meanings, tags, phrases, sentences, relations, created_at, updated_at
FROM words
WHERE lower(text) = lower($1) AND language = $2
ORDER BY
  CASE WHEN word_type = 'lemma' THEN 0 ELSE 1 END,
  id
LIMIT 1;

-- name: ListWords :many
SELECT id, text, language, word_type, lemma, phonetics, meanings, tags, phrases, sentences, relations, created_at, updated_at
FROM words
WHERE (sqlc.arg(language_filter) = '' OR language = sqlc.arg(language_filter))
  AND (sqlc.arg(keyword_filter) = '' OR text ILIKE sqlc.arg(keyword_filter) || '%')
  AND (sqlc.arg(word_type_filter) = '' OR word_type = sqlc.arg(word_type_filter))
  AND (
        COALESCE(array_length(sqlc.arg(words_filter)::text[], 1), 0) = 0
        OR lower(text) = ANY(sqlc.arg(words_filter)::text[])
      )
ORDER BY
  CASE
    WHEN sqlc.arg(keyword_filter) <> '' AND lower(text) = lower(sqlc.arg(keyword_filter)) THEN 0
    ELSE 1
  END,
  CASE WHEN sqlc.arg(primary_key) = 'created_at' AND sqlc.arg(primary_desc) THEN created_at END DESC NULLS LAST,
  CASE WHEN sqlc.arg(primary_key) = 'created_at' AND NOT sqlc.arg(primary_desc) THEN created_at END ASC NULLS LAST,
  CASE WHEN sqlc.arg(primary_key) = 'updated_at' AND sqlc.arg(primary_desc) THEN updated_at END DESC NULLS LAST,
  CASE WHEN sqlc.arg(primary_key) = 'updated_at' AND NOT sqlc.arg(primary_desc) THEN updated_at END ASC NULLS LAST,
  CASE WHEN sqlc.arg(primary_key) = 'text' AND sqlc.arg(primary_desc) THEN text END DESC NULLS LAST,
  CASE WHEN sqlc.arg(primary_key) = 'text' AND NOT sqlc.arg(primary_desc) THEN text END ASC NULLS LAST,
  CASE WHEN sqlc.arg(primary_key) = 'id' AND sqlc.arg(primary_desc) THEN id END DESC,
  CASE WHEN sqlc.arg(primary_key) = 'id' AND NOT sqlc.arg(primary_desc) THEN id END ASC,
  CASE WHEN sqlc.arg(secondary_key) = 'created_at' AND sqlc.arg(secondary_desc) THEN created_at END DESC NULLS LAST,
  CASE WHEN sqlc.arg(secondary_key) = 'created_at' AND NOT sqlc.arg(secondary_desc) THEN created_at END ASC NULLS LAST,
  CASE WHEN sqlc.arg(secondary_key) = 'updated_at' AND sqlc.arg(secondary_desc) THEN updated_at END DESC NULLS LAST,
  CASE WHEN sqlc.arg(secondary_key) = 'updated_at' AND NOT sqlc.arg(secondary_desc) THEN updated_at END ASC NULLS LAST,
  CASE WHEN sqlc.arg(secondary_key) = 'text' AND sqlc.arg(secondary_desc) THEN text END DESC NULLS LAST,
  CASE WHEN sqlc.arg(secondary_key) = 'text' AND NOT sqlc.arg(secondary_desc) THEN text END ASC NULLS LAST,
  CASE WHEN sqlc.arg(secondary_key) = 'id' AND sqlc.arg(secondary_desc) THEN id END DESC,
  CASE WHEN sqlc.arg(secondary_key) = 'id' AND NOT sqlc.arg(secondary_desc) THEN id END ASC,
  id ASC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- name: ListWordsByPOS :many
SELECT id, text, language, word_type, lemma, phonetics, meanings, tags, phrases, sentences, relations, created_at, updated_at
FROM words
WHERE meanings @> $1::jsonb
ORDER BY text ASC
LIMIT $2 OFFSET $3;

-- List all variants/inflections for a lemma
-- name: ListInflections :many
SELECT id, text, language, word_type, lemma, phonetics, meanings, tags, phrases, sentences, relations, created_at, updated_at
FROM words
WHERE language = $1 AND lemma = $2
ORDER BY text ASC;
