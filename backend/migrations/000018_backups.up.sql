-- Manual tenant backup registry. Each row points to a compressed dump file
-- stored in the backup directory.
CREATE TABLE backups (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL REFERENCES tenants (id) ON DELETE RESTRICT,
    created_by   uuid REFERENCES users (id) ON DELETE SET NULL,
    file_name    varchar(255) NOT NULL,
    size_bytes   bigint NOT NULL,
    status       varchar(20) NOT NULL DEFAULT 'completed',
    table_count  integer NOT NULL DEFAULT 0,
    row_count    bigint NOT NULL DEFAULT 0,
    notes        text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_backups_status CHECK (status IN ('completed', 'failed')),
    CONSTRAINT chk_backups_size CHECK (size_bytes >= 0),
    CONSTRAINT chk_backups_table_count CHECK (table_count >= 0),
    CONSTRAINT chk_backups_row_count CHECK (row_count >= 0)
);

CREATE INDEX idx_backups_tenant_time ON backups (tenant_id, created_at DESC);
