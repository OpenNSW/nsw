-- ============================================================================
-- Seed: Customs House Agents (CHA)
-- ============================================================================

INSERT INTO customs_house_agents (id, name, description, email)
VALUES
	('a1b2c3d4-0001-4000-8000-000000000001', 'ADAM PVT LTD', 'user having trader and cha roles', 'user123@abcd-traders.private-sector.dev'),
	('a1b2c3d4-0002-4000-8000-000000000002', 'ACME NUTRITION PVT LTD', 'user having only cha role', 'user456@abcd-traders.private-sector.dev'),
	('a1b2c3d4-0003-4000-8000-000000000003', 'ADVANTIS PROJECTS LTD', 'Advantis Projects - Offers experienced clearance services', NULL),
	('a1b2c3d4-0004-4000-8000-000000000004', 'YUSEN LOGISTICS LTD', 'Yusen - Global logistics and customs', NULL),
	('a1b2c3d4-0005-4000-8000-000000000005', 'MALSHIP CUSTOMS LTD', 'Malship - Shipping and customs house agency', NULL)
ON CONFLICT (id) DO UPDATE SET email = EXCLUDED.email;
