-- Append-only audit trail. No updated_at/deleted_at by design.
CREATE TABLE audit_logs (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id   uuid REFERENCES tenants (id) ON DELETE RESTRICT,
    user_id     uuid,
    actor_name  varchar(120),
    action      varchar(60) NOT NULL,
    module      varchar(40) NOT NULL,
    entity_type varchar(60) NOT NULL,
    entity_id   uuid,
    before_data jsonb,
    after_data  jsonb,
    ip          inet,
    user_agent  varchar(255),
    request_id  varchar(64),
    created_at  timestamptz NOT NULL DEFAULT now()
);
