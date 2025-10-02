-- 0001_words.down.sql
-- Rollback initial schema

DROP INDEX IF EXISTS idx_words_meanings_gin;
DROP INDEX IF EXISTS idx_words_text_prefix;
DROP INDEX IF EXISTS idx_words_lemma;
DROP INDEX IF EXISTS idx_words_language_lower_text;
ALTER TABLE words DROP CONSTRAINT IF EXISTS chk_words_lemma_ref;
DROP TABLE IF EXISTS words;
