-- Máquina de estados dos pedidos de catálogo (creator_orders) + campos billing.

ALTER TABLE creator_orders DROP CONSTRAINT IF EXISTS creator_orders_status_check;

UPDATE creator_orders SET status = 'requested' WHERE status = 'pending';

ALTER TABLE creator_orders
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS billing_payment_ref TEXT;

ALTER TABLE creator_orders
    ALTER COLUMN status SET DEFAULT 'requested';

ALTER TABLE creator_orders ADD CONSTRAINT creator_orders_status_check
    CHECK (status IN (
        'requested',
        'awaiting_payment',
        'paid',
        'fulfilled',
        'canceled',
        'refunded'
    ));

CREATE UNIQUE INDEX IF NOT EXISTS idx_creator_orders_billing_payment_ref_unique
    ON creator_orders (billing_payment_ref)
    WHERE billing_payment_ref IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_creator_orders_buyer_created
    ON creator_orders (buyer_id, created_at DESC);
