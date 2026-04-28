-- Migration: add locale + notify_preferences to users (HB-AUTH-04 §6.3)

ALTER TABLE IF EXISTS users
    ADD COLUMN IF NOT EXISTS locale TEXT DEFAULT 'pt-BR',
    ADD COLUMN IF NOT EXISTS notify_preferences JSONB DEFAULT '{"email": true, "push": true}'::jsonb;
