-- Platform tenants and their outlets.
CREATE TABLE tenants (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name          varchar(150) NOT NULL,
    slug          varchar(60)  NOT NULL,
    status        varchar(20)  NOT NULL DEFAULT 'active',
    plan          varchar(20)  NOT NULL DEFAULT 'free',
    trial_ends_at timestamptz,
    owner_name    varchar(150),
    owner_phone   varchar(30),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_tenants_slug UNIQUE (slug),
    CONSTRAINT chk_tenants_status CHECK (status IN ('active', 'suspended', 'trial')),
    CONSTRAINT chk_tenants_plan CHECK (plan IN ('free', 'starter', 'pro'))
);

CREATE TABLE outlets (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants (id) ON DELETE RESTRICT,
    code       varchar(20)  NOT NULL,
    name       varchar(120) NOT NULL,
    address    text,
    phone      varchar(30),
    timezone   varchar(50)  NOT NULL DEFAULT 'Asia/Jakarta',
    is_active  boolean      NOT NULL DEFAULT true,
    created_at timestamptz  NOT NULL DEFAULT now(),
    updated_at timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT uq_outlets_tenant_code UNIQUE (tenant_id, code)
);
