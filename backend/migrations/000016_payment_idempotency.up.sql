-- Idempotency keys make retried payment requests safe without creating a
-- second payment. The key is scoped per tenant and only set when supplied.
ALTER TABLE payments ADD COLUMN idempotency_key varchar(100);

CREATE UNIQUE INDEX uq_payments_tenant_idempotency
    ON payments (tenant_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;
