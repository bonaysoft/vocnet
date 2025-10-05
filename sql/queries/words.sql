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
WHERE (COALESCE(sqlc.arg('language')::text, '') = '' OR language = sqlc.arg('language'))
  AND (COALESCE(sqlc.arg('keyword')::text, '') = '' OR text ILIKE COALESCE(sqlc.arg('keyword')::text, '') || '%')
  AND (COALESCE(sqlc.arg('word_type')::text, '') = '' OR word_type = sqlc.arg('word_type'))
  AND (
        COALESCE(array_length(sqlc.arg('words')::text[], 1), 0) = 0
        OR lower(text) = ANY(sqlc.arg('words')::text[])
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
WHERE (COALESCE(sqlc.arg('language')::text, '') = '' OR language = sqlc.arg('language'))
  AND (COALESCE(sqlc.arg('keyword')::text, '') = '' OR text ILIKE COALESCE(sqlc.arg('keyword')::text, '') || '%')
  AND (COALESCE(sqlc.arg('word_type')::text, '') = '' OR word_type = sqlc.arg('word_type'))
  AND (
        COALESCE(array_length(sqlc.arg('words')::text[], 1), 0) = 0
        OR lower(text) = ANY(sqlc.arg('words')::text[])
      )
ORDER BY
  CASE
    WHEN COALESCE(sqlc.arg('keyword')::text, '') <> '' AND lower(text) = lower(COALESCE(sqlc.arg('keyword')::text, '')) THEN 0
    ELSE 1
  END,
  CASE WHEN COALESCE(sqlc.arg('primary_key')::text, '') = 'created_at' AND COALESCE(sqlc.arg('primary_desc')::bool, false) THEN created_at END DESC NULLS LAST,
  CASE WHEN COALESCE(sqlc.arg('primary_key')::text, '') = 'created_at' AND NOT COALESCE(sqlc.arg('primary_desc')::bool, false) THEN created_at END ASC NULLS LAST,
  CASE WHEN COALESCE(sqlc.arg('primary_key')::text, '') = 'updated_at' AND COALESCE(sqlc.arg('primary_desc')::bool, false) THEN updated_at END DESC NULLS LAST,
  CASE WHEN COALESCE(sqlc.arg('primary_key')::text, '') = 'updated_at' AND NOT COALESCE(sqlc.arg('primary_desc')::bool, false) THEN updated_at END ASC NULLS LAST,
  CASE WHEN COALESCE(sqlc.arg('primary_key')::text, '') = 'text' AND COALESCE(sqlc.arg('primary_desc')::bool, false) THEN text END DESC NULLS LAST,
  CASE WHEN COALESCE(sqlc.arg('primary_key')::text, '') = 'text' AND NOT COALESCE(sqlc.arg('primary_desc')::bool, false) THEN text END ASC NULLS LAST,
  CASE WHEN COALESCE(sqlc.arg('primary_key')::text, '') = 'id' AND COALESCE(sqlc.arg('primary_desc')::bool, false) THEN id END DESC,
  CASE WHEN COALESCE(sqlc.arg('primary_key')::text, '') = 'id' AND NOT COALESCE(sqlc.arg('primary_desc')::bool, false) THEN id END ASC,
  CASE WHEN COALESCE(sqlc.arg('secondary_key')::text, '') = 'created_at' AND COALESCE(sqlc.arg('secondary_desc')::bool, false) THEN created_at END DESC NULLS LAST,
  CASE WHEN COALESCE(sqlc.arg('secondary_key')::text, '') = 'created_at' AND NOT COALESCE(sqlc.arg('secondary_desc')::bool, false) THEN created_at END ASC NULLS LAST,
  CASE WHEN COALESCE(sqlc.arg('secondary_key')::text, '') = 'updated_at' AND COALESCE(sqlc.arg('secondary_desc')::bool, false) THEN updated_at END DESC NULLS LAST,
  CASE WHEN COALESCE(sqlc.arg('secondary_key')::text, '') = 'updated_at' AND NOT COALESCE(sqlc.arg('secondary_desc')::bool, false) THEN updated_at END ASC NULLS LAST,
  CASE WHEN COALESCE(sqlc.arg('secondary_key')::text, '') = 'text' AND COALESCE(sqlc.arg('secondary_desc')::bool, false) THEN text END DESC NULLS LAST,
  CASE WHEN COALESCE(sqlc.arg('secondary_key')::text, '') = 'text' AND NOT COALESCE(sqlc.arg('secondary_desc')::bool, false) THEN text END ASC NULLS LAST,
  CASE WHEN COALESCE(sqlc.arg('secondary_key')::text, '') = 'id' AND COALESCE(sqlc.arg('secondary_desc')::bool, false) THEN id END DESC,
  CASE WHEN COALESCE(sqlc.arg('secondary_key')::text, '') = 'id' AND NOT COALESCE(sqlc.arg('secondary_desc')::bool, false) THEN id END ASC,
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
