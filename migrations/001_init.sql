CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TYPE user_role AS ENUM ('admin', 'partner');
CREATE TYPE service_type AS ENUM ('finkoper', 'whatsapp');
CREATE TYPE payment_state AS ENUM ('active', 'cancelled');
CREATE TYPE renewal_state AS ENUM ('active', 'cancelled');

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name TEXT NOT NULL,
    role user_role NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL UNIQUE,
    csrf_token TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ip_address INET,
    user_agent TEXT
);

CREATE TABLE clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_name TEXT NOT NULL,
    tax_id TEXT,
    email TEXT NOT NULL UNIQUE,
    contact_name TEXT,
    phone TEXT,
    comment TEXT,
    archived_at TIMESTAMPTZ,
    version INTEGER NOT NULL DEFAULT 1,
    created_by UUID NOT NULL REFERENCES users(id),
    updated_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id),
    service service_type NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    enabled_at DATE,
    expires_at DATE,
    version INTEGER NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (client_id, service),
    CHECK (service <> 'whatsapp' OR enabled = FALSE OR enabled_at IS NOT NULL)
);

CREATE TABLE monthly_client_counts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id),
    month DATE NOT NULL CHECK (month = date_trunc('month', month)::date),
    count INTEGER CHECK (count >= 0),
    version INTEGER NOT NULL DEFAULT 1,
    updated_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (client_id, month)
);

CREATE TABLE monthly_confirmations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    month DATE NOT NULL UNIQUE CHECK (month = date_trunc('month', month)::date),
    confirmed_by UUID NOT NULL REFERENCES users(id),
    confirmed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revision INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id),
    service service_type NOT NULL,
    received_on DATE NOT NULL,
    billing_month DATE NOT NULL CHECK (billing_month = date_trunc('month', billing_month)::date),
    months_count INTEGER NOT NULL CHECK (months_count > 0 AND months_count <= 120),
    amount NUMERIC(14,2) CHECK (amount IS NULL OR amount >= 0),
    currency CHAR(3) NOT NULL DEFAULT 'KZT',
    comment TEXT,
    state payment_state NOT NULL DEFAULT 'active',
    cancellation_reason TEXT,
    cancelled_at TIMESTAMPTZ,
    cancelled_by UUID REFERENCES users(id),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    idempotency_key UUID NOT NULL UNIQUE
);

CREATE TABLE payment_allocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL REFERENCES payments(id),
    billing_month DATE NOT NULL CHECK (billing_month = date_trunc('month', billing_month)::date),
    UNIQUE (payment_id, billing_month)
);

CREATE TABLE renewals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    service service_type NOT NULL,
    previous_expires_at DATE,
    new_expires_at DATE NOT NULL,
    months_count INTEGER NOT NULL CHECK (months_count > 0 AND months_count <= 120),
    payment_id UUID REFERENCES payments(id),
    comment TEXT,
    state renewal_state NOT NULL DEFAULT 'active',
    cancellation_reason TEXT,
    cancelled_at TIMESTAMPTZ,
    cancelled_by UUID REFERENCES users(id),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    idempotency_key UUID NOT NULL UNIQUE
);

CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    client_id UUID REFERENCES clients(id),
    user_id UUID NOT NULL REFERENCES users(id),
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    old_values JSONB,
    new_values JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX clients_company_trgm_idx ON clients USING gin (company_name gin_trgm_ops);
CREATE INDEX clients_tax_id_idx ON clients (tax_id);
CREATE INDEX clients_phone_idx ON clients (phone);
CREATE INDEX counts_month_idx ON monthly_client_counts (month, client_id);
CREATE INDEX payments_client_month_idx ON payments (client_id, service, billing_month);
CREATE INDEX allocations_month_idx ON payment_allocations (billing_month, payment_id);
CREATE INDEX renewals_client_created_idx ON renewals (client_id, created_at DESC);
CREATE UNIQUE INDEX renewals_active_payment_idx ON renewals (payment_id) WHERE payment_id IS NOT NULL AND state = 'active';
CREATE INDEX audit_client_created_idx ON audit_logs (client_id, created_at DESC);
CREATE INDEX sessions_expires_idx ON sessions (expires_at);

-- Финансовые записи нельзя удалить: исправления выполняются только отменой.
CREATE OR REPLACE FUNCTION prevent_financial_delete() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'financial records cannot be deleted; cancel them instead';
END $$;
CREATE TRIGGER payments_no_delete BEFORE DELETE ON payments FOR EACH ROW EXECUTE FUNCTION prevent_financial_delete();
CREATE TRIGGER renewals_no_delete BEFORE DELETE ON renewals FOR EACH ROW EXECUTE FUNCTION prevent_financial_delete();
CREATE TRIGGER audit_no_update BEFORE UPDATE OR DELETE ON audit_logs FOR EACH ROW EXECUTE FUNCTION prevent_financial_delete();
