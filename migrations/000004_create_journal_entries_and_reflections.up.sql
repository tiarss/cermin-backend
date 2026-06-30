CREATE TABLE journal_entries (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
    title VARCHAR(255),
    content TEXT NOT NULL,
    is_crisis BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_journal_entries_user_id ON journal_entries(user_id);
CREATE INDEX idx_journal_entries_user_created_at ON journal_entries(user_id, created_at DESC);

CREATE TABLE journal_reflections (
    id BIGSERIAL PRIMARY KEY,
    journal_entry_id BIGINT NOT NULL UNIQUE REFERENCES journal_entries(id) ON UPDATE CASCADE ON DELETE CASCADE,
    validation TEXT,
    growth_prompt TEXT,
    dominant_emotion VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_journal_reflections_entry_id ON journal_reflections(journal_entry_id);

CREATE TABLE reflection_summaries (
    id BIGSERIAL PRIMARY KEY,
    reflection_id BIGINT NOT NULL REFERENCES journal_reflections(id) ON UPDATE CASCADE ON DELETE CASCADE,
    position INTEGER NOT NULL,
    summary_text TEXT NOT NULL,
    CONSTRAINT uq_reflection_summaries_position UNIQUE (reflection_id, position)
);

CREATE INDEX idx_reflection_summaries_reflection_id ON reflection_summaries(reflection_id);

CREATE TABLE reflection_hidden_languages (
    id BIGSERIAL PRIMARY KEY,
    reflection_id BIGINT NOT NULL REFERENCES journal_reflections(id) ON UPDATE CASCADE ON DELETE CASCADE,
    position INTEGER NOT NULL,
    hidden_language TEXT NOT NULL,
    CONSTRAINT uq_reflection_hidden_languages_position UNIQUE (reflection_id, position)
);

CREATE INDEX idx_reflection_hidden_languages_reflection_id ON reflection_hidden_languages(reflection_id);

CREATE TABLE reflection_emotion_scores (
    id BIGSERIAL PRIMARY KEY,
    reflection_id BIGINT NOT NULL REFERENCES journal_reflections(id) ON UPDATE CASCADE ON DELETE CASCADE,
    label VARCHAR(50) NOT NULL,
    percentage NUMERIC(5,2) NOT NULL,
    CONSTRAINT uq_reflection_emotion_scores_label UNIQUE (reflection_id, label),
    CONSTRAINT chk_reflection_emotion_percentage CHECK (percentage >= 0 AND percentage <= 100)
);

CREATE INDEX idx_reflection_emotion_scores_reflection_id ON reflection_emotion_scores(reflection_id);
