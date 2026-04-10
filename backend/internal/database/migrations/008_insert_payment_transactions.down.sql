-- Migration: 008_insert_payment_transactions.down.sql
-- Description: Revert payment_transactions table creation.

DROP TABLE IF EXISTS payment_transactions;
