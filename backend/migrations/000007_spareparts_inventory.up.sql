-- Spareparts, suppliers, and append-only inventory movements.
CREATE TABLE sparepart_categories (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants (id) ON DELETE RESTRICT,
    name       varchar(80) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_sparepart_categories_tenant_name UNIQUE (tenant_id, name)
);

CREATE TABLE suppliers (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL REFERENCES tenants (id) ON DELETE RESTRICT,
    name         varchar(120) NOT NULL,
    contact_name varchar(120),
    phone        varchar(30),
    address      text,
    notes        text,
    deleted_at   timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE spareparts (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         uuid NOT NULL REFERENCES tenants (id) ON DELETE RESTRICT,
    category_id       uuid REFERENCES sparepart_categories (id) ON DELETE SET NULL,
    supplier_id       uuid REFERENCES suppliers (id) ON DELETE SET NULL,
    sku               varchar(40) NOT NULL,
    barcode           varchar(60),
    name              varchar(150) NOT NULL,
    compatible_models text,
    stock             integer NOT NULL DEFAULT 0,
    min_stock         integer NOT NULL DEFAULT 0,
    cost_price        bigint NOT NULL DEFAULT 0,
    sell_price        bigint NOT NULL DEFAULT 0,
    location          varchar(60),
    image_path        varchar(255),
    is_active         boolean NOT NULL DEFAULT true,
    last_stock_at     timestamptz,
    deleted_at        timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_spareparts_stock CHECK (stock >= 0),
    CONSTRAINT chk_spareparts_min_stock CHECK (min_stock >= 0),
    CONSTRAINT chk_spareparts_cost_price CHECK (cost_price >= 0),
    CONSTRAINT chk_spareparts_sell_price CHECK (sell_price >= 0)
);

CREATE TABLE inventory_movements (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      uuid NOT NULL REFERENCES tenants (id) ON DELETE RESTRICT,
    outlet_id      uuid NOT NULL REFERENCES outlets (id) ON DELETE RESTRICT,
    sparepart_id   uuid NOT NULL REFERENCES spareparts (id) ON DELETE RESTRICT,
    movement_type  varchar(25) NOT NULL,
    qty            integer NOT NULL,
    stock_before   integer NOT NULL,
    stock_after    integer NOT NULL,
    unit_cost      bigint NOT NULL DEFAULT 0,
    reference_type varchar(30) NOT NULL,
    reference_id   uuid,
    notes          text,
    created_by     uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_inventory_movements_type CHECK (movement_type IN (
        'initial', 'purchase_in', 'service_out', 'service_return',
        'adjustment_in', 'adjustment_out', 'transfer_in', 'transfer_out'
    )),
    CONSTRAINT chk_inventory_movements_qty CHECK (qty <> 0),
    CONSTRAINT chk_inventory_movements_stock_before CHECK (stock_before >= 0),
    CONSTRAINT chk_inventory_movements_stock_after CHECK (stock_after >= 0),
    CONSTRAINT chk_inventory_movements_unit_cost CHECK (unit_cost >= 0),
    CONSTRAINT chk_inventory_movements_reference_type
        CHECK (reference_type IN ('service_order', 'purchase', 'opname', 'transfer'))
);
