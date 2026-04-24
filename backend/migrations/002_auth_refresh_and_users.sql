-- Migration: refresh token hashing + user verification fields
-- HB-AUTH-01 — opaque refresh with rotation, bcrypt hash, email verification support

-- Rename sessions → refresh_tokens (matches spec naming)
ALTER TABLE IF EXISTS sessions RENAME TO refresh_tokens;

-- Rename old plaintext column so we can drop it cleanly
ALTER TABLE IF EXISTS refresh_tokens RENAME COLUMN refresh_token TO legacy_token;

-- New columns for secure opaque token storage
ALTER TABLE IF EXISTS refresh_tokens
    ADD COLUMN IF NOT EXISTS token_id UUID,
    ADD COLUMN IF NOT EXISTS token_hash TEXT,
    ADD COLUMN IF NOT EXISTS last_used_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS ip_address INET,
    ADD COLUMN IF NOT EXISTS user_agent TEXT;

-- Populate token_id for any existing rows (old plaintext tokens become invalid after hash migration anyway)
UPDATE refresh_tokens SET token_id = gen_random_uuid() WHERE token_id IS NULL;

-- Make token_id non-null after population
ALTER TABLE IF EXISTS refresh_tokens ALTER COLUMN token_id SET NOT NULL;

-- Add unique constraint for token lookup
ALTER TABLE IF EXISTS refresh_tokens
    ADD CONSTRAINT refresh_tokens_token_id_unique UNIQUE (token_id);

-- Drop the legacy plaintext column — old tokens are now invalid (would need re-auth)
ALTER TABLE IF EXISTS refresh_tokens DROP COLUMN IF EXISTS legacy_token;

-- User fields: email verification, terms acceptance, password rotation tracking
ALTER TABLE IF EXISTS users
    ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS accept_terms_version TEXT,
    ADD COLUMN IF NOT EXISTS password_changed_at TIMESTAMPTZ;

-- Drop old sessions index, create new refresh_tokens indexes
DROP INDEX IF EXISTS idx_sessions_refresh;

-- Active tokens per user (for global revocation / LogoutAll)
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_active
    ON refresh_tokens(user_id) WHERE revoked_at IS NULL;

-- Token lookup during refresh (unique token_id already indexed, but partial index for hot path)
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_lookup
    ON refresh_tokens(token_id) WHERE revoked_at IS NULL;

-- Cleanup candidates: expired + revoked tokens (optional periodic DELETE)
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_cleanup
    ON refresh_tokens(expires_at) WHERE revoked_at IS NOT NULL;
