CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL,
    currency VARCHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    balance BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0),
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'SUSPENDED', 'CLOSED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reference_id VARCHAR(64) NOT NULL UNIQUE,
    transaction_type VARCHAR(30) NOT NULL CHECK (transaction_type IN ('WALLET_TRANSFER')),
    amount BIGINT NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    status VARCHAR(20) NOT NULL CHECK (status IN ('PENDING','COMPLETED','FAILED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    CHECK ((status = 'COMPLETED' AND completed_at IS NOT NULL) OR status <> 'COMPLETED')
);

CREATE TABLE IF NOT EXISTS postings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE RESTRICT,
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE RESTRICT,
    amount BIGINT NOT NULL CHECK (amount > 0),
    direction VARCHAR(6) NOT NULL CHECK (direction IN ('DEBIT', 'CREDIT')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(transaction_id, direction),
    UNIQUE(transaction_id, wallet_id, direction)
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    idempotency_key UUID PRIMARY KEY,
    payload_hash CHAR(64) NOT NULL,
    response_status INTEGER,
    response_body JSONB,
    status VARCHAR(20) NOT NULL CHECK (status IN ('PROCESSING','COMPLETED','FAILED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '24 hours'),
    CHECK ((status = 'COMPLETED' AND response_status IS NOT NULL AND response_body IS NOT NULL) OR status <> 'COMPLETED')
);

-- indexes
CREATE INDEX idx_wallets_owner_id ON wallets(owner_id);
CREATE INDEX idx_transactions_created_at ON transactions(created_at DESC, id DESC);
CREATE INDEX idx_postings_wallet_created ON postings(wallet_id, created_at DESC, id DESC);
CREATE INDEX idx_postings_transaction ON postings(transaction_id);
CREATE INDEX idx_idempotency_expires_at ON idempotency_keys(expires_at);
