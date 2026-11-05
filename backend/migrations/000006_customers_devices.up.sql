-- Customers and their devices.
CREATE TABLE customers (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        uuid NOT NULL REFERENCES tenants (id) ON DELETE RESTRICT,
    name             varchar(120) NOT NULL,
    phone            varchar(30) NOT NULL,
    phone_normalized varchar(20) NOT NULL,
    address          text,
    notes            text,
    last_service_at  timestamptz,
    deleted_at       timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    created_by       uuid REFERENCES users (id) ON DELETE SET NULL,
    updated_by       uuid REFERENCES users (id) ON DELETE SET NULL
);

CREATE TABLE devices (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants (id) ON DELETE RESTRICT,
    customer_id uuid NOT NULL REFERENCES customers (id) ON DELETE RESTRICT,
    brand       varchar(60) NOT NULL,
    model       varchar(100) NOT NULL,
    imei        varchar(20),
    color       varchar(40),
    notes       text,
    deleted_at  timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
