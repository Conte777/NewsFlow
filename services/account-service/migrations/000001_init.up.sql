-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create accounts table for storing Telegram account metadata
CREATE TABLE IF NOT EXISTS accounts (
    id SERIAL PRIMARY KEY,
    phone_number VARCHAR(32) NOT NULL UNIQUE,
    phone_hash VARCHAR(64) NOT NULL UNIQUE,
    status VARCHAR(32) NOT NULL DEFAULT 'inactive',
    last_connected_at TIMESTAMP,
    last_error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create sessions table for storing MTProto session data
CREATE TABLE IF NOT EXISTS sessions (
    id SERIAL PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    session_data BYTEA NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_session_account UNIQUE (account_id)
);

-- Create indexes for better query performance
CREATE INDEX idx_accounts_phone_hash ON accounts(phone_hash);
CREATE INDEX idx_accounts_status ON accounts(status);
CREATE INDEX idx_sessions_account_id ON sessions(account_id);

-- Create triggers for automatic updated_at updates
CREATE TRIGGER update_accounts_updated_at
    BEFORE UPDATE ON accounts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_sessions_updated_at
    BEFORE UPDATE ON sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
