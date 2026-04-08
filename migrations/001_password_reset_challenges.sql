-- Recuperação de senha (código + JWT de reset). Legado: coluna telefone.
-- Após 002_email_password_reset.sql a tabela é recriada com email_norm.
-- Executar uma vez no PostgreSQL da API.

CREATE TABLE IF NOT EXISTS password_reset_challenges (
  id BIGSERIAL PRIMARY KEY,
  telefone TEXT NOT NULL,
  code_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  attempt_count INT NOT NULL DEFAULT 0,
  consumed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_password_reset_tel_active
  ON password_reset_challenges (telefone, id DESC)
  WHERE consumed_at IS NULL;
