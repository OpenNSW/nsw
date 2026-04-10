-- Migration: 002_insert_seed_hscodes.down.sql
-- Description: Roll back HS code seed data.

DELETE FROM hs_codes 
WHERE hs_code IN ('4bdfb1f0-2b71-4ddc-8b99-f31c3d7660bc', '9d5b7a1e-4c2f-4a0b-9d5b-7a1e4c2f4a0b');
