-- Enable required extension for UUID generation
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Customers
CREATE TABLE IF NOT EXISTS customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    subscription_tier VARCHAR(50) DEFAULT 'starter',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- API Keys
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    key_name VARCHAR(100) NOT NULL,
    key_prefix VARCHAR(20) NOT NULL,
    key_hash VARCHAR(64) NOT NULL UNIQUE,
    key_last_four VARCHAR(4) NOT NULL,
    environment VARCHAR(20) DEFAULT 'test',
    permissions JSONB DEFAULT '{"mfa":{"register":true,"validate":true}}',
    rate_limit_per_hour INTEGER DEFAULT 10000,
    last_used_at TIMESTAMPTZ,
    usage_count BIGINT DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- MFA Users
CREATE TABLE IF NOT EXISTS mfa_users (
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    secret_key_encrypted TEXT NOT NULL,
    backup_codes_encrypted TEXT[],
    account_name VARCHAR(255),
    issuer VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    api_key_id UUID REFERENCES api_keys(id),
    PRIMARY KEY (customer_id, user_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_mfa_users_customer_user ON mfa_users(customer_id, user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_customer ON api_keys(customer_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(is_active);
