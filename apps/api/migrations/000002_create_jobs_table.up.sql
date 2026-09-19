-- Migration 000002: Criação da tabela jobs para armazenamento e indexação de vagas
CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    external_id VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL,
    company VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    city VARCHAR(100) NOT NULL DEFAULT '',
    state VARCHAR(10) NOT NULL DEFAULT '',
    source VARCHAR(100) NOT NULL,
    source_url TEXT NOT NULL,
    fingerprint CHAR(64) NOT NULL,
    work_mode VARCHAR(30) NOT NULL DEFAULT '',
    employment_type VARCHAR(30) NOT NULL DEFAULT '',
    salary_min BIGINT,
    salary_max BIGINT,
    published_at TIMESTAMPTZ,
    collected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_jobs_source_external_id UNIQUE (source, external_id)
);

CREATE INDEX IF NOT EXISTS idx_jobs_title ON jobs (title);
CREATE INDEX IF NOT EXISTS idx_jobs_city ON jobs (city);
CREATE INDEX IF NOT EXISTS idx_jobs_company ON jobs (company);
CREATE INDEX IF NOT EXISTS idx_jobs_published_at ON jobs (published_at DESC NULLS LAST, id DESC);
CREATE INDEX IF NOT EXISTS idx_jobs_fingerprint ON jobs (fingerprint);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs (status);
CREATE INDEX IF NOT EXISTS idx_jobs_last_seen_at ON jobs (last_seen_at);
