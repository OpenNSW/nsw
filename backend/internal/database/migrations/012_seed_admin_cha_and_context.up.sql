-- ============================================================================
-- Migration: 00012_seed_admin_cha_and_context.sql
-- Purpose: Final seed for admin account and CHA assignment context.
-- ============================================================================

INSERT INTO customs_house_agents (id, name, description, email)
VALUES ('c3d4e5f6-7890-4000-8000-000000000001', 'Admin CHA', 'Default admin clearing house agent', 'admin@thunder.dev')
ON CONFLICT (id) DO NOTHING;

INSERT INTO user_contexts (user_id, user_context)
VALUES ('admin@thunder.dev', '{"role": "cha", "cha_id": "c3d4e5f6-7890-4000-8000-000000000001"}')
ON CONFLICT (user_id) DO NOTHING;
