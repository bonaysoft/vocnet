CREATE TABLE IF NOT EXISTS user_words (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    word TEXT NOT NULL,
    word_normalized TEXT GENERATED ALWAYS AS (lower(word)) STORED,
    language TEXT NOT NULL DEFAULT 'en',
    mastery_listen SMALLINT NOT NULL DEFAULT 0,
    mastery_read SMALLINT NOT NULL DEFAULT 0,
    mastery_spell SMALLINT NOT NULL DEFAULT 0,
    mastery_pronounce SMALLINT NOT NULL DEFAULT 0,
    mastery_use SMALLINT NOT NULL DEFAULT 0,
    mastery_overall INTEGER NOT NULL DEFAULT 0,
    review_last_review_at TIMESTAMPTZ,
    review_next_review_at TIMESTAMPTZ,
    review_interval_days INTEGER NOT NULL DEFAULT 0,
    review_fail_count INTEGER NOT NULL DEFAULT 0,
    query_count BIGINT NOT NULL DEFAULT 0,
    notes TEXT,
    sentences JSONB NOT NULL DEFAULT '[]'::jsonb,
    relations JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_user_words_user_word UNIQUE (user_id, word_normalized)
);

CREATE INDEX IF NOT EXISTS idx_user_words_user ON user_words(user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_words_word_lower ON user_words(word_normalized);
