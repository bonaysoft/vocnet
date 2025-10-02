-- 0001_words.up.sql
-- Initial schema for words table (denormalized lexical entries)

CREATE TABLE IF NOT EXISTS words (
    id BIGSERIAL PRIMARY KEY,           -- 自增ID (仅用于基础CRUD, 不用于关联)
    text TEXT NOT NULL,
    language TEXT NOT NULL DEFAULT 'en',
    word_type TEXT NOT NULL DEFAULT 'lemma',
    lemma TEXT NULL,
    phonetics JSONB NOT NULL DEFAULT '[]'::jsonb,
    meanings JSONB NOT NULL DEFAULT '[]'::jsonb,
    tags TEXT[] NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_words_lang_text_type UNIQUE (language, text, word_type)
);

ALTER TABLE words
    ADD CONSTRAINT chk_words_lemma_ref CHECK (
        (word_type = 'lemma' AND lemma IS NULL) OR (word_type <> 'lemma' AND lemma IS NOT NULL)
    );

-- Indexes
CREATE INDEX IF NOT EXISTS idx_words_lemma ON words(language, lemma) WHERE lemma IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_words_text_prefix ON words (text varchar_pattern_ops);
CREATE INDEX IF NOT EXISTS idx_words_meanings_gin ON words USING GIN (meanings jsonb_path_ops);
CREATE INDEX IF NOT EXISTS idx_words_language_lower_text ON words (language, lower(text));
