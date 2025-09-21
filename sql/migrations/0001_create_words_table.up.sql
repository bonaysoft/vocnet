-- 0001_create_words_table.up.sql
CREATE TABLE IF NOT EXISTS words (
    id BIGSERIAL PRIMARY KEY,
    lemma TEXT NOT NULL UNIQUE,
    language TEXT NOT NULL DEFAULT 'en',
    phonetic TEXT NULL,
    pos TEXT NULL,
    definition TEXT NULL,
    translation TEXT NULL,
    exchange TEXT NULL,
    tags TEXT[] NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_words_language_lemma ON words(language, lemma);
