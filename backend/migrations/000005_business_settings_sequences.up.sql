-- Per-tenant business settings and atomic document numbering.
CREATE TABLE business_settings (
    id                           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                    uuid NOT NULL REFERENCES tenants (id) ON DELETE RESTRICT,
    outlet_id                    uuid REFERENCES outlets (id) ON DELETE CASCADE,
    store_name                   varchar(150) NOT NULL,
    legal_name                   varchar(150),
    logo_path                    varchar(255),
    address                      text,
    phone                        varchar(30),
    email                        varchar(150),
    tax_id                       varchar(30),
    currency                     char(3) NOT NULL DEFAULT 'IDR',
    timezone                     varchar(50) NOT NULL DEFAULT 'Asia/Jakarta',
    ticket_prefix                varchar(10) NOT NULL DEFAULT 'FP',
    ticket_format                varchar(60) NOT NULL DEFAULT '{PREFIX}-{YYMMDD}-{SEQ:4}',
    invoice_prefix               varchar(10) NOT NULL DEFAULT 'INV',
    invoice_format               varchar(60) NOT NULL DEFAULT '{PREFIX}-{YYMMDD}-{SEQ:4}',
    invoice_terms                text,
    invoice_footer               text,
    default_service_warranty_days integer NOT NULL DEFAULT 30,
    default_part_warranty_days    integer NOT NULL DEFAULT 90,
    receipt_paper_width          smallint NOT NULL DEFAULT 58,
    technician_sees_only_assigned boolean NOT NULL DEFAULT true,
    public_tracking_enabled      boolean NOT NULL DEFAULT true,
    wa_provider                  varchar(20) NOT NULL DEFAULT 'noop',
    wa_config_enc                bytea,
    wa_templates                 jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at                   timestamptz NOT NULL DEFAULT now(),
    updated_at                   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_business_settings_tenant_outlet
        UNIQUE NULLS NOT DISTINCT (tenant_id, outlet_id),
    CONSTRAINT chk_business_settings_currency CHECK (currency = 'IDR'),
    CONSTRAINT chk_business_settings_service_warranty CHECK (default_service_warranty_days >= 0),
    CONSTRAINT chk_business_settings_part_warranty CHECK (default_part_warranty_days >= 0),
    CONSTRAINT chk_business_settings_paper_width CHECK (receipt_paper_width IN (58, 80))
);

CREATE TABLE document_sequences (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants (id) ON DELETE RESTRICT,
    outlet_id   uuid NOT NULL REFERENCES outlets (id) ON DELETE CASCADE,
    doc_type    varchar(30) NOT NULL,
    period_key  varchar(10) NOT NULL,
    last_number integer NOT NULL DEFAULT 0,
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_document_sequences_key UNIQUE (tenant_id, outlet_id, doc_type, period_key),
    CONSTRAINT chk_document_sequences_doc_type
        CHECK (doc_type IN ('service_order', 'invoice', 'payment', 'expense', 'warranty', 'claim')),
    CONSTRAINT chk_document_sequences_last_number CHECK (last_number >= 0)
);
