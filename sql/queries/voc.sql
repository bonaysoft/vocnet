-- Single-table JSONB queries for vocs

-- name: CreateVoc :one
INSERT INTO vocs (text, language, voc_type, lemma, phonetic, meanings, tags)
VALUES ($1,$2,$3,$4,$5,$6,$7)
RETURNING id, text, language, voc_type, lemma, phonetic, meanings, tags, created_at;

-- Get specific voc by composite key
-- name: GetVoc :one
SELECT id, text, language, voc_type, lemma, phonetic, meanings, tags, created_at
FROM vocs
WHERE language = $1 AND text = $2 AND voc_type = $3
LIMIT 1;

-- Lookup any voc entry by text (prefer lemma). We return the lemma row if exists, else any row.
-- name: LookupVoc :one
WITH candidates AS (
    SELECT *, CASE WHEN voc_type='lemma' THEN 0 ELSE 1 END AS priority
    FROM vocs
    WHERE lower(text) = lower($1) AND language = $2
)
SELECT id, text, language, voc_type, lemma, phonetic, meanings, tags, created_at
FROM candidates
ORDER BY priority ASC
LIMIT 1;

-- name: ListVocs :many
SELECT id, text, language, voc_type, lemma, phonetic, meanings, tags, created_at
FROM vocs
WHERE ($1::text IS NULL OR language = $1)
  AND ($2::text IS NULL OR text ILIKE $2 || '%')
  AND ($3::text IS NULL OR voc_type = $3)
ORDER BY text ASC
LIMIT $4 OFFSET $5;

-- name: ListVocsByPOS :many
SELECT id, text, language, voc_type, lemma, phonetic, meanings, tags, created_at
FROM vocs
WHERE meanings @> $1::jsonb
ORDER BY text ASC
LIMIT $2 OFFSET $3;

-- List all variants/inflections for a lemma
-- name: ListInflections :many
SELECT id, text, language, voc_type, lemma, phonetic, meanings, tags, created_at
FROM vocs
WHERE language = $1 AND lemma = $2
ORDER BY text ASC;
