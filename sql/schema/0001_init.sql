-- Single-table lexical schema (denormalized) with JSONB columns
-- Stores all POS entries, senses, translations, examples, notes, forms inside one row.
-- This favors simpler write & retrieval at cost of larger row size & reduced granular querying.

CREATE TABLE IF NOT EXISTS vocs (
    id BIGSERIAL PRIMARY KEY,           -- 自增ID (仅用于基础CRUD, 不用于关联)
    text TEXT NOT NULL,                 -- 单词或其变形本身
    language TEXT NOT NULL DEFAULT 'en',
    voc_type TEXT NOT NULL DEFAULT 'lemma', -- 词类型: lemma, past, past_participle, present_participle, third_person_singular, plural, comparative, superlative, variant, derived, other
    lemma TEXT NULL,                    -- 若本行是变形, 指向其原形(与 language 一起定位); 若 voc_type='lemma' 则为空
    phonetic TEXT NULL,
    meanings JSONB NOT NULL DEFAULT '[]'::jsonb,  -- Array of {pos, definition, translation} (只在 lemma 行一般有值)
    tags TEXT[] NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_vocs_lang_text_type UNIQUE (language, text, voc_type)
);

-- 约束: lemma 行不能有 lemma 值; 非 lemma 行必须有 lemma
-- Use plain ADD CONSTRAINT for sqlc compatibility (migration guards existence)
ALTER TABLE vocs
    ADD CONSTRAINT chk_vocs_lemma_ref CHECK (
        (voc_type = 'lemma' AND lemma IS NULL) OR (voc_type <> 'lemma' AND lemma IS NOT NULL)
    );

-- 为 lemma 查询添加索引 (language, lemma) 用于从原形找所有变形
CREATE INDEX IF NOT EXISTS idx_vocs_lemma ON vocs(language, lemma) WHERE lemma IS NOT NULL;

-- 前缀检索
CREATE INDEX IF NOT EXISTS idx_vocs_text_prefix ON vocs (text varchar_pattern_ops);

-- meanings JSONB 索引
CREATE INDEX IF NOT EXISTS idx_vocs_meanings_gin ON vocs USING GIN (meanings jsonb_path_ops);


