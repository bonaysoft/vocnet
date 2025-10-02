ALTER TABLE words
    ADD COLUMN IF NOT EXISTS phrases TEXT[] NULL,
    ADD COLUMN IF NOT EXISTS sentences JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS relations JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE words
SET
    sentences = COALESCE(sentences, '[]'::jsonb),
    relations = COALESCE(relations, '[]'::jsonb),
    updated_at = COALESCE(updated_at, NOW());
