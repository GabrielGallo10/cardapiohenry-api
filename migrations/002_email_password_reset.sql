-- E-mail nos utilizadores + recuperação de senha por e-mail (Brevo).
-- Executar depois de 001_password_reset_challenges.sql.

ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS email VARCHAR(320);

CREATE UNIQUE INDEX IF NOT EXISTS idx_usuarios_email_lower
  ON usuarios (LOWER(TRIM(email)))
  WHERE email IS NOT NULL AND LENGTH(TRIM(email)) > 0;

DROP INDEX IF EXISTS idx_password_reset_tel_active;
DROP TABLE IF EXISTS password_reset_challenges;

CREATE TABLE password_reset_challenges (
  id BIGSERIAL PRIMARY KEY,
  email_norm TEXT NOT NULL,
  code_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  attempt_count INT NOT NULL DEFAULT 0,
  consumed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_password_reset_email_active
  ON password_reset_challenges (email_norm, id DESC)
  WHERE consumed_at IS NULL;
