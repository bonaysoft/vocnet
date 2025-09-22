-- 0001_init.up.sql
-- Initial schema for vocs table (denormalized lexical entries)

CREATE TABLE IF NOT EXISTS vocs (
    id BIGSERIAL PRIMARY KEY,           -- 自增ID (仅用于基础CRUD, 不用于关联)
    text TEXT NOT NULL,
    language TEXT NOT NULL DEFAULT 'en',
    voc_type TEXT NOT NULL DEFAULT 'lemma',
    lemma TEXT NULL,
    phonetic TEXT NULL,
    meanings JSONB NOT NULL DEFAULT '[]'::jsonb,
    tags TEXT[] NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_vocs_lang_text_type UNIQUE (language, text, voc_type)
);

ALTER TABLE vocs
    ADD CONSTRAINT chk_vocs_lemma_ref CHECK (
        (voc_type = 'lemma' AND lemma IS NULL) OR (voc_type <> 'lemma' AND lemma IS NOT NULL)
    );

-- Indexes
CREATE INDEX IF NOT EXISTS idx_vocs_lemma ON vocs(language, lemma) WHERE lemma IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_vocs_text_prefix ON vocs (text varchar_pattern_ops);
CREATE INDEX IF NOT EXISTS idx_vocs_meanings_gin ON vocs USING GIN (meanings jsonb_path_ops);
CREATE INDEX IF NOT EXISTS idx_vocs_language_lower_text ON vocs (language, lower(text));
