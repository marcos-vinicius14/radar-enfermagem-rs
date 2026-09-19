-- Migration inicial para verificação de conectividade e integridade do banco
CREATE TABLE IF NOT EXISTS _system_meta (
    key VARCHAR(100) PRIMARY KEY,
    val TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO _system_meta (key, val)
VALUES ('system_version', '1.0.0')
ON CONFLICT (key) DO NOTHING;
