-- Migration: OAuth accounts + PKCE state storage (HB-AUTH-05)

-- External identity linkage: one oauth account per (provider, subject).
-- Allows merge: same email across providers maps to same user.
CREATE TABLE IF NOT EXISTS oauth_accounts (
    id          TEXT PRIMARY KEY,                       -- provider '|' subject (e.g. "google|12345")
    user_id     UUID NOT NULL REFERENCES users(id),
    provider    TEXT NOT NULL,                          -- "google", "github", etc.
    subject     TEXT NOT NULL,                          -- sub claim from ID token
    email       TEXT,                                   -- email from provider at link time
    display_name TEXT,
    avatar_url  TEXT,
    linked_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, subject)
);

CREATE INDEX IF NOT EXISTS idx_oauth_accounts_user_id ON oauth_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_accounts_email ON oauth_accounts(email);

-- Short-lived PKCE + CSRF state for OAuth authorize→callback flow.
-- Rows are deleted after use or after expiry.
CREATE TABLE IF NOT EXISTS oauth_states (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider        TEXT NOT NULL,
    code_verifier   TEXT NOT NULL,          -- PKCE code_verifier (S256)
    nonce           TEXT NOT NULL,          -- CSRF nonce (stateID.nonce → nonce validated server-side)
    redirect_uri    TEXT NOT NULL,          -- expected callback URL
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '10 minutes')
);

CREATE INDEX IF NOT EXISTS idx_oauth_states_expires ON oauth_states(expires_at);
