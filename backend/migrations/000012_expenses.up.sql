-- Operational expenses.
CREATE TABLE expense_categories (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants (id) ON DELETE RESTRICT,
    name       varchar(80) NOT NULL,
    is_system  boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_expense_categories_tenant_name UNIQUE (tenant_id, name)
);

CREATE TABLE expenses (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      uuid NOT NULL REFERENCES tenants (id) ON DELETE RESTRICT,
    outlet_id      uuid NOT NULL REFERENCES outlets (id) ON DELETE RESTRICT,
    expense_no     varchar(40) NOT NULL,
    category_id    uuid NOT NULL REFERENCES expense_categories (id) ON DELETE RESTRICT,
    description    varchar(200) NOT NULL,
    amount         bigint NOT NULL,
    expense_date   date NOT NULL,
    method         varchar(20) NOT NULL DEFAULT 'cash',
    receipt_path   varchar(255),
    notes          text,
    deleted_at     timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_expenses_tenant_no UNIQUE (tenant_id, expense_no),
    CONSTRAINT chk_expenses_amount CHECK (amount > 0),
    CONSTRAINT chk_expenses_method CHECK (method IN ('cash', 'bank_transfer', 'qris', 'other'))
);
