CREATE TABLE magic_codes (
    id UUID PRIMARY KEY,
    email_normalized TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX magic_codes_email_normalized_idx ON magic_codes(email_normalized);
