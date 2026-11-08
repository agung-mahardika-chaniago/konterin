DROP INDEX IF EXISTS uq_payments_tenant_idempotency;
ALTER TABLE payments DROP COLUMN IF EXISTS idempotency_key;
