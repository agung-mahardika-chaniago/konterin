-- In-app notifications and the WhatsApp outbox.
CREATE TABLE notifications (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants (id) ON DELETE RESTRICT,
    user_id     uuid REFERENCES users (id) ON DELETE CASCADE,
    type        varchar(40) NOT NULL,
    title       varchar(150) NOT NULL,
    body        text,
    link        varchar(255),
    entity_type varchar(60),
    entity_id   uuid,
    is_read     boolean NOT NULL DEFAULT false,
    read_at     timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE whatsapp_messages (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           uuid NOT NULL REFERENCES tenants (id) ON DELETE RESTRICT,
    outlet_id           uuid NOT NULL REFERENCES outlets (id) ON DELETE RESTRICT,
    service_order_id    uuid REFERENCES service_orders (id) ON DELETE SET NULL,
    customer_id         uuid REFERENCES customers (id) ON DELETE SET NULL,
    phone               varchar(30) NOT NULL,
    template_key        varchar(50) NOT NULL,
    payload             jsonb NOT NULL DEFAULT '{}'::jsonb,
    body                text,
    status              varchar(15) NOT NULL DEFAULT 'queued',
    attempts            smallint NOT NULL DEFAULT 0,
    last_error          text,
    provider            varchar(20) NOT NULL DEFAULT 'noop',
    provider_message_id varchar(100),
    scheduled_at        timestamptz NOT NULL DEFAULT now(),
    sent_at             timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_whatsapp_messages_status
        CHECK (status IN ('queued', 'processing', 'sent', 'failed', 'cancelled')),
    CONSTRAINT chk_whatsapp_messages_attempts CHECK (attempts >= 0)
);
