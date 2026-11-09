-- Stage 9: public tracking, warranty/claim rework linkage, notification read
-- receipts, and idempotency keys for notification/WhatsApp jobs.

-- Tenant broadcast notifications (user_id IS NULL) need a per-user read state.
CREATE TABLE notification_reads (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id uuid NOT NULL REFERENCES notifications (id) ON DELETE CASCADE,
    user_id         uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    read_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_notification_reads_user UNIQUE (notification_id, user_id)
);

CREATE INDEX idx_notification_reads_user ON notification_reads (user_id, read_at DESC);

ALTER TABLE notifications ADD COLUMN dedupe_key varchar(120);

CREATE UNIQUE INDEX uq_notifications_tenant_dedupe
    ON notifications (tenant_id, dedupe_key)
    WHERE dedupe_key IS NOT NULL;

ALTER TABLE whatsapp_messages ADD COLUMN dedupe_key varchar(120);

CREATE UNIQUE INDEX uq_whatsapp_messages_tenant_dedupe
    ON whatsapp_messages (tenant_id, dedupe_key)
    WHERE dedupe_key IS NOT NULL;

-- Rework orders keep an explicit link back to the original order and claim.
ALTER TABLE service_orders ADD COLUMN order_type varchar(15) NOT NULL DEFAULT 'normal';
ALTER TABLE service_orders ADD COLUMN rework_of_service_order_id uuid
    REFERENCES service_orders (id) ON DELETE SET NULL;
ALTER TABLE service_orders ADD COLUMN warranty_claim_id uuid
    REFERENCES warranty_claims (id) ON DELETE SET NULL;

ALTER TABLE service_orders ADD CONSTRAINT chk_service_orders_order_type
    CHECK (order_type IN ('normal', 'rework'));

CREATE INDEX idx_service_orders_rework_of
    ON service_orders (tenant_id, rework_of_service_order_id);
CREATE INDEX idx_service_orders_warranty_claim
    ON service_orders (tenant_id, warranty_claim_id);
