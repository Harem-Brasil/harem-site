-- Assinatura por criador (subscribers): coluna referenciada em FeedHome / posts.service.
-- Planos só com plan_id continuam com creator_id NULL até o billing preencher por criador.

ALTER TABLE subscriptions
    ADD COLUMN IF NOT EXISTS creator_id UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_creator_active
    ON subscriptions (user_id, creator_id)
    WHERE status IN ('active', 'trialing') AND creator_id IS NOT NULL;
