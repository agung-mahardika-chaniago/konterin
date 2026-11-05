-- Refresh-token sessions with rotation and reuse detection support.
CREATE TABLE refresh_tokens (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash     char(64) NOT NULL,
    family_id      uuid NOT NULL,
    expires_at     timestamptz NOT NULL,
    revoked_at     timestamptz,
    replaced_by_id uuid REFERENCES refresh_tokens (id) ON DELETE SET NULL,
    ip             inet,
    user_agent     varchar(255),
    created_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_refresh_tokens_token_hash UNIQUE (token_hash)
);
