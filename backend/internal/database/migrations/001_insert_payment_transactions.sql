CREATE TABLE IF NOT EXISTS payment_transactions (
    id text NOT NULL PRIMARY KEY,
    reference_number VARCHAR(255) UNIQUE NOT NULL,
    task_id VARCHAR(255) NOT NULL,
    session_id VARCHAR(255),
    amount NUMERIC(15, 2) NOT NULL,
    currency VARCHAR(10) NOT NULL,
    status VARCHAR(50) NOT NULL,
    payment_method VARCHAR(50),
    expiry_date TIMESTAMP WITH TIME ZONE NOT NULL,
    gateway_metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_payment_tx_task_id ON payment_transactions (task_id);
CREATE INDEX idx_payment_tx_reference_number ON payment_transactions (reference_number);
