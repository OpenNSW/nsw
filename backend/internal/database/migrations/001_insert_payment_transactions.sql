-- ============================================================================
-- Migration: Create payment_transactions table for Payment Mock
-- ============================================================================
CREATE TABLE IF NOT EXISTS payment_transactions (
    id uuid NOT NULL PRIMARY KEY,
    task_id uuid NOT NULL,
    execution_id varchar(100) NOT NULL,
    reference_number varchar(100) NOT NULL UNIQUE,
    provider_id varchar(50) NOT NULL,
    status varchar(50) NOT NULL DEFAULT 'PENDING'
        CONSTRAINT payment_transactions_status_check
            CHECK ((status)::text = ANY ((ARRAY['PENDING'::character varying, 'COMPLETED'::character varying, 'FAILED'::character varying])::text[])),
    amount numeric(15, 2) NOT NULL,
    currency varchar(10) NOT NULL DEFAULT 'LKR',
    payer_name varchar(255),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

COMMENT ON TABLE payment_transactions IS 'Tracks payment transactions for tasks';
COMMENT ON COLUMN payment_transactions.task_id IS 'ID of the task associated with the payment';
COMMENT ON COLUMN payment_transactions.execution_id IS 'ID of the FSM execution associated with the payment';
COMMENT ON COLUMN payment_transactions.reference_number IS 'Unique reference number for external payment verification';
COMMENT ON COLUMN payment_transactions.status IS 'Status of the payment: PENDING, COMPLETED, or FAILED';
COMMENT ON COLUMN payment_transactions.amount IS 'Amount of the payment';

CREATE INDEX IF NOT EXISTS idx_payment_transactions_task_id ON payment_transactions (task_id);
CREATE INDEX IF NOT EXISTS idx_payment_transactions_execution_id ON payment_transactions (execution_id);
