-- Catálogo de itens do criador e pedidos associados.

CREATE TABLE IF NOT EXISTS creator_catalog (
    id UUID PRIMARY KEY,
    creator_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price_cents INT NOT NULL CHECK (price_cents >= 0),
    currency TEXT NOT NULL DEFAULT 'BRL',
    visibility TEXT NOT NULL DEFAULT 'public' CHECK (visibility IN ('public', 'subscribers', 'premium')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS creator_orders (
    id UUID PRIMARY KEY,
    creator_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    buyer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_id UUID NOT NULL REFERENCES creator_catalog(id) ON DELETE RESTRICT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'canceled', 'refunded', 'fulfilled')),
    amount_cents INT NOT NULL CHECK (amount_cents >= 0),
    currency TEXT NOT NULL DEFAULT 'BRL',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_creator_catalog_creator_created
    ON creator_catalog (creator_id, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_creator_orders_creator_created
    ON creator_orders (creator_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_creator_orders_item
    ON creator_orders (item_id);
