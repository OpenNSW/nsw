-- ============================================================================
-- Customs House Agents (CHA) and two-stage consignment support
-- ============================================================================

-- Table: customs_house_agents
CREATE TABLE IF NOT EXISTS customs_house_agents
(
	id         uuid      DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
	name       varchar(255)                        NOT NULL,
	description text,
	created_at timestamptz DEFAULT now()           NOT NULL,
	updated_at timestamptz DEFAULT now()           NOT NULL
);

COMMENT ON TABLE customs_house_agents IS 'Clearing House Agents / Customs House Agents for consignment assignment';

-- Add cha_id to consignments (nullable for backward compatibility)
ALTER TABLE consignments
	ADD COLUMN IF NOT EXISTS cha_id uuid REFERENCES customs_house_agents (id);

COMMENT ON COLUMN consignments.cha_id IS 'Assigned Customs House Agent (CHA); set at Stage 1 by Trader';

-- Allow AWAITING_INITIATION state (Stage 1 shell before CHA selects HS Code)
ALTER TABLE consignments
	DROP CONSTRAINT IF EXISTS consignments_state_check;

ALTER TABLE consignments
	ADD CONSTRAINT consignments_state_check
		CHECK ((state)::text = ANY (ARRAY['AWAITING_INITIATION'::character varying, 'IN_PROGRESS'::character varying, 'FINISHED'::character varying]));

-- Index for CHA-filtered list
CREATE INDEX IF NOT EXISTS idx_consignments_cha_id
	ON consignments (cha_id);

-- Seed: 5 Major Service Providers (fixed UUIDs for idempotent re-runs)
INSERT INTO customs_house_agents (id, name, description)
VALUES
	('a1b2c3d4-0001-4000-8000-000000000001', 'Spectra', 'Spectra Logistics - Customs clearance and freight'),
	('a1b2c3d4-0002-4000-8000-000000000002', 'Aitken Spence', 'Aitken Spence - Integrated logistics and agency services'),
	('a1b2c3d4-0003-4000-8000-000000000003', 'Advantis', 'Advantis Projects - Offers experienced clearance services'),
	('a1b2c3d4-0004-4000-8000-000000000004', 'Yusen', 'Yusen - Global logistics and customs'),
	('a1b2c3d4-0005-4000-8000-000000000005', 'Malship', 'Malship - Shipping and customs house agency')
ON CONFLICT (id) DO NOTHING;
