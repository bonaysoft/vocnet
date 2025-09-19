-- 002_vocab.sql
-- Vocabulary related tables

-- Core word table (global lexeme) for future extension; optional now
CREATE TABLE IF NOT EXISTS words (
    id BIGSERIAL PRIMARY KEY,
    lemma VARCHAR(128) NOT NULL,              -- canonical form
    language VARCHAR(8) NOT NULL DEFAULT 'en',
    phonetic VARCHAR(128),
    pos VARCHAR(32),                          -- primary part of speech
    definition TEXT,                          -- optional primary definition
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(language, lemma)
);

-- User specific word record (personal learning context)
CREATE TABLE IF NOT EXISTS user_words (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    word_id BIGINT REFERENCES words(id) ON DELETE SET NULL,
    custom_text VARCHAR(128),                 -- For ad-hoc word not in words table
    status VARCHAR(16) NOT NULL DEFAULT 'unknown',  -- unknown|learning|familiar|mastered|reinforced|relearn
    mastery_listen SMALLINT NOT NULL DEFAULT 0 CHECK (mastery_listen BETWEEN 0 AND 5),
    mastery_read SMALLINT NOT NULL DEFAULT 0 CHECK (mastery_read BETWEEN 0 AND 5),
    mastery_spell SMALLINT NOT NULL DEFAULT 0 CHECK (mastery_spell BETWEEN 0 AND 5),
    mastery_pronounce SMALLINT NOT NULL DEFAULT 0 CHECK (mastery_pronounce BETWEEN 0 AND 5),
    mastery_use SMALLINT NOT NULL DEFAULT 0 CHECK (mastery_use BETWEEN 0 AND 5),
    mastery_overall SMALLINT NOT NULL DEFAULT 0 CHECK (mastery_overall BETWEEN 0 AND 500), -- store *100
    last_review_at TIMESTAMPTZ,
    next_review_at TIMESTAMPTZ,
    review_interval_days INT NOT NULL DEFAULT 0,
    review_fail_count INT NOT NULL DEFAULT 0,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, word_id),
    UNIQUE(user_id, custom_text)
);
CREATE INDEX IF NOT EXISTS idx_user_words_user ON user_words(user_id);
CREATE INDEX IF NOT EXISTS idx_user_words_next_review ON user_words(user_id, next_review_at) WHERE next_review_at IS NOT NULL;

-- Mastery history
CREATE TABLE IF NOT EXISTS word_mastery_history (
    id BIGSERIAL PRIMARY KEY,
    user_word_id BIGINT NOT NULL REFERENCES user_words(id) ON DELETE CASCADE,
    dimension VARCHAR(16) NOT NULL, -- listen|read|spell|pronounce|use|overall
    old_value SMALLINT NOT NULL,
    new_value SMALLINT NOT NULL,
    delta SMALLINT NOT NULL,
    trigger_type VARCHAR(16) NOT NULL, -- manual|test|listening|spelling|algorithm|import
    context JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_word_mastery_history_user_word ON word_mastery_history(user_word_id);

-- Word relations
CREATE TABLE IF NOT EXISTS word_relations (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    word_a_id BIGINT NOT NULL REFERENCES user_words(id) ON DELETE CASCADE,
    word_b_id BIGINT NOT NULL REFERENCES user_words(id) ON DELETE CASCADE,
    relation_type VARCHAR(32) NOT NULL,
    subtype VARCHAR(32),
    is_bidirectional BOOLEAN NOT NULL DEFAULT FALSE,
    weight SMALLINT NOT NULL DEFAULT 50 CHECK (weight BETWEEN 1 AND 100),
    note TEXT,
    created_source VARCHAR(16) NOT NULL DEFAULT 'manual',
    metadata JSONB,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (word_a_id <> word_b_id),
    UNIQUE(user_id, word_a_id, word_b_id, relation_type)
);
CREATE INDEX IF NOT EXISTS idx_word_relations_user ON word_relations(user_id);
CREATE INDEX IF NOT EXISTS idx_word_relations_type ON word_relations(user_id, relation_type);

-- Relation clusters
CREATE TABLE IF NOT EXISTS relation_clusters (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    type VARCHAR(16) NOT NULL DEFAULT 'custom', -- theme|mnemonic|set|custom
    color VARCHAR(12),
    icon VARCHAR(32),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, name)
);

CREATE TABLE IF NOT EXISTS relation_cluster_members (
    cluster_id BIGINT NOT NULL REFERENCES relation_clusters(id) ON DELETE CASCADE,
    user_word_id BIGINT NOT NULL REFERENCES user_words(id) ON DELETE CASCADE,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY(cluster_id, user_word_id)
);

-- Sentences
CREATE TABLE IF NOT EXISTS sentences (
    id BIGSERIAL PRIMARY KEY,
    content TEXT NOT NULL,
    content_norm TEXT NOT NULL,
    language VARCHAR(8) NOT NULL DEFAULT 'en',
    hash CHAR(40) NOT NULL UNIQUE,
    length INT NOT NULL,
    token_count INT,
    created_source VARCHAR(16) NOT NULL DEFAULT 'manual',
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sentences_language ON sentences(language);

-- User sentences
CREATE TABLE IF NOT EXISTS user_sentences (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sentence_id BIGINT NOT NULL REFERENCES sentences(id) ON DELETE CASCADE,
    is_starred BOOLEAN NOT NULL DEFAULT FALSE,
    familiarity SMALLINT NOT NULL DEFAULT 0 CHECK (familiarity BETWEEN 0 AND 5),
    last_review_at TIMESTAMPTZ,
    next_review_at TIMESTAMPTZ,
    review_interval_days INT NOT NULL DEFAULT 0,
    private_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, sentence_id)
);
CREATE INDEX IF NOT EXISTS idx_user_sentences_user ON user_sentences(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sentences_next_review ON user_sentences(user_id, next_review_at) WHERE next_review_at IS NOT NULL;

-- Sources
CREATE TABLE IF NOT EXISTS sources (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(16) NOT NULL DEFAULT 'other',
    title VARCHAR(255),
    author VARCHAR(128),
    url TEXT,
    reference TEXT,
    tag_list TEXT[],
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sources_user ON sources(user_id);

CREATE TABLE IF NOT EXISTS sentence_sources (
    sentence_id BIGINT NOT NULL REFERENCES sentences(id) ON DELETE CASCADE,
    source_id BIGINT NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    relation_type VARCHAR(16) NOT NULL DEFAULT 'direct',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY(sentence_id, source_id)
);

-- Word usages
CREATE TABLE IF NOT EXISTS word_usages (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_word_id BIGINT NOT NULL REFERENCES user_words(id) ON DELETE CASCADE,
    sentence_id BIGINT NOT NULL REFERENCES sentences(id) ON DELETE CASCADE,
    start_offset INT NOT NULL,
    end_offset INT NOT NULL,
    original_text TEXT NOT NULL,
    normalized_form TEXT NOT NULL,
    grammatical_role VARCHAR(32),
    usage_type VARCHAR(16) NOT NULL DEFAULT 'base',
    confidence SMALLINT NOT NULL DEFAULT 100 CHECK (confidence BETWEEN 0 AND 100),
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (start_offset >= 0 AND end_offset > start_offset)
);
CREATE INDEX IF NOT EXISTS idx_word_usages_user_word ON word_usages(user_id, user_word_id);
CREATE INDEX IF NOT EXISTS idx_word_usages_sentence ON word_usages(sentence_id);
CREATE INDEX IF NOT EXISTS idx_word_usages_norm ON word_usages(normalized_form);

