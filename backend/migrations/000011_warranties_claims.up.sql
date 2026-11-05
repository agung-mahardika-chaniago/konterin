-- After-sales warranties and claims.
CREATE TABLE warranties (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        uuid NOT NULL REFERENCES tenants (id) ON DELETE RESTRICT,
    outlet_id        uuid NOT NULL REFERENCES outlets (id) ON DELETE RESTRICT,
    service_order_id uuid NOT NULL REFERENCES service_orders (id) ON DELETE CASCADE,
    invoice_id       uuid NOT NULL REFERENCES invoices (id) ON DELETE RESTRICT,
    customer_id      uuid NOT NULL REFERENCES customers (id) ON DELETE RESTRICT,
    warranty_no      varchar(40) NOT NULL,
    type             varchar(10) NOT NULL,
    start_date       date NOT NULL,
    end_date         date NOT NULL,
    days             integer NOT NULL,
    terms            text,
    status           varchar(10) NOT NULL DEFAULT 'active',
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_warranties_tenant_no UNIQUE (tenant_id, warranty_no),
    CONSTRAINT uq_warranties_service_order UNIQUE (service_order_id),
    CONSTRAINT chk_warranties_type CHECK (type IN ('service', 'part', 'both')),
    CONSTRAINT chk_warranties_status CHECK (status IN ('active', 'expired', 'void')),
    CONSTRAINT chk_warranties_days CHECK (days >= 0),
    CONSTRAINT chk_warranties_dates CHECK (end_date >= start_date)
);

CREATE TABLE warranty_claims (
    id                      uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               uuid NOT NULL REFERENCES tenants (id) ON DELETE RESTRICT,
    outlet_id               uuid NOT NULL REFERENCES outlets (id) ON DELETE RESTRICT,
    warranty_id             uuid NOT NULL REFERENCES warranties (id) ON DELETE RESTRICT,
    claim_no                varchar(40) NOT NULL,
    complaint               text NOT NULL,
    status                  varchar(15) NOT NULL DEFAULT 'submitted',
    resolution              text,
    rework_service_order_id uuid REFERENCES service_orders (id) ON DELETE SET NULL,
    handled_by              uuid REFERENCES users (id) ON DELETE SET NULL,
    claimed_at              timestamptz NOT NULL DEFAULT now(),
    resolved_at             timestamptz,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_warranty_claims_tenant_no UNIQUE (tenant_id, claim_no),
    CONSTRAINT chk_warranty_claims_status
        CHECK (status IN ('submitted', 'approved', 'rejected', 'in_progress', 'completed'))
);
