ALTER TABLE words
    DROP COLUMN IF EXISTS relations,
    DROP COLUMN IF EXISTS sentences,
    DROP COLUMN IF EXISTS phrases,
    DROP COLUMN IF EXISTS updated_at;
