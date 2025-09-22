-- 0001_init.down.sql
-- Rollback initial schema

DROP INDEX IF EXISTS idx_vocs_meanings_gin;
DROP INDEX IF EXISTS idx_vocs_text_prefix;
DROP INDEX IF EXISTS idx_vocs_lemma;
DROP INDEX IF EXISTS idx_vocs_language_lower_text;
ALTER TABLE vocs DROP CONSTRAINT IF EXISTS chk_vocs_lemma_ref;
DROP TABLE IF EXISTS vocs;
