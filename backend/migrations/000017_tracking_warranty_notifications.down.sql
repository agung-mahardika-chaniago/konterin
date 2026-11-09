DROP INDEX IF EXISTS idx_service_orders_warranty_claim;
DROP INDEX IF EXISTS idx_service_orders_rework_of;

ALTER TABLE service_orders DROP CONSTRAINT IF EXISTS chk_service_orders_order_type;
ALTER TABLE service_orders DROP COLUMN IF EXISTS warranty_claim_id;
ALTER TABLE service_orders DROP COLUMN IF EXISTS rework_of_service_order_id;
ALTER TABLE service_orders DROP COLUMN IF EXISTS order_type;

DROP INDEX IF EXISTS uq_whatsapp_messages_tenant_dedupe;
ALTER TABLE whatsapp_messages DROP COLUMN IF EXISTS dedupe_key;

DROP INDEX IF EXISTS uq_notifications_tenant_dedupe;
ALTER TABLE notifications DROP COLUMN IF EXISTS dedupe_key;

DROP INDEX IF EXISTS idx_notification_reads_user;
DROP TABLE IF EXISTS notification_reads;
