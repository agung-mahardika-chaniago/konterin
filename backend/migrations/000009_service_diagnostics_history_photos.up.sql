-- Diagnosis, immutable status history, and service photos.
CREATE TABLE service_diagnostics (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         uuid NOT NULL REFERENCES tenants (id) ON DELETE RESTRICT,
    service_order_id  uuid NOT NULL REFERENCES service_orders (id) ON DELETE CASCADE,
    diagnosis         text,
    findings          text,
    action_taken      text,
    test_result       text,
    technician_notes  text,
    diagnosed_by      uuid REFERENCES users (id) ON DELETE SET NULL,
    diagnosed_at      timestamptz,
    estimated_done_at timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_service_diagnostics_order UNIQUE (service_order_id)
);

CREATE TABLE service_status_history (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        uuid NOT NULL REFERENCES tenants (id) ON DELETE RESTRICT,
    service_order_id uuid NOT NULL REFERENCES service_orders (id) ON DELETE CASCADE,
    from_status      varchar(30),
    to_status        varchar(30) NOT NULL,
    notes            text,
    changed_by       uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_service_status_history_from CHECK (
        from_status IS NULL OR from_status IN (
            'received', 'waiting_diagnosis', 'waiting_approval', 'waiting_part',
            'in_progress', 'testing', 'completed', 'ready_for_pickup',
            'picked_up', 'cancelled'
        )
    ),
    CONSTRAINT chk_service_status_history_to CHECK (to_status IN (
        'received', 'waiting_diagnosis', 'waiting_approval', 'waiting_part',
        'in_progress', 'testing', 'completed', 'ready_for_pickup',
        'picked_up', 'cancelled'
    ))
);

CREATE TABLE service_photos (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        uuid NOT NULL REFERENCES tenants (id) ON DELETE RESTRICT,
    service_order_id uuid NOT NULL REFERENCES service_orders (id) ON DELETE CASCADE,
    stage            varchar(15) NOT NULL,
    file_path        varchar(255) NOT NULL,
    mime_type        varchar(50) NOT NULL,
    size_bytes       integer NOT NULL DEFAULT 0,
    is_public        boolean NOT NULL DEFAULT false,
    uploaded_by      uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_service_photos_stage CHECK (stage IN ('intake', 'before', 'after', 'testing', 'other')),
    CONSTRAINT chk_service_photos_size CHECK (size_bytes >= 0)
);
