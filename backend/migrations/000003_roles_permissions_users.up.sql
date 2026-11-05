-- Identity and RBAC foundation.
CREATE TABLE roles (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug       varchar(30) NOT NULL,
    name       varchar(60) NOT NULL,
    is_system  boolean     NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_roles_slug UNIQUE (slug)
);

CREATE TABLE permissions (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code        varchar(80) NOT NULL,
    module      varchar(40) NOT NULL,
    description varchar(200),
    CONSTRAINT uq_permissions_code UNIQUE (code)
);

CREATE TABLE role_permissions (
    role_id       uuid NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    permission_id uuid NOT NULL REFERENCES permissions (id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE users (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            uuid REFERENCES tenants (id) ON DELETE RESTRICT,
    role_id              uuid NOT NULL REFERENCES roles (id) ON DELETE RESTRICT,
    name                 varchar(120) NOT NULL,
    email                varchar(150) NOT NULL,
    phone                varchar(30),
    password_hash        varchar(100) NOT NULL,
    avatar_path          varchar(255),
    is_active            boolean      NOT NULL DEFAULT true,
    failed_login_count   smallint     NOT NULL DEFAULT 0,
    locked_until         timestamptz,
    last_login_at        timestamptz,
    permission_version   integer      NOT NULL DEFAULT 1,
    must_change_password boolean      NOT NULL DEFAULT false,
    deleted_at           timestamptz,
    created_at           timestamptz  NOT NULL DEFAULT now(),
    updated_at           timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT chk_users_failed_login_count CHECK (failed_login_count >= 0),
    CONSTRAINT chk_users_permission_version CHECK (permission_version > 0)
);

CREATE TABLE user_outlets (
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    outlet_id  uuid NOT NULL REFERENCES outlets (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, outlet_id)
);
