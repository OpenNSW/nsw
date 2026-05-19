-- ============================================================================
-- Seed: Customs House Agents (CHA)
-- ============================================================================

INSERT INTO customs_house_agents (id, name, description, email, company_id)
VALUES
	('a1b2c3d4-0001-4000-8000-000000000001', 'Suresh', 'User with Trader and CHA roles at ADAM PVT LTD',   'suresh@adam-pvt-ltd.private-sector.dev',  'adam-pvt-ltd'),
	('a1b2c3d4-0002-4000-8000-000000000002', 'Ramesh', 'User with CHA role at ADAM PVT LTD',              'ramesh@adam-pvt-ltd.private-sector.dev',  'adam-pvt-ltd'),
	('a1b2c3d4-0003-4000-8000-000000000003', 'Naresh', 'User with CHA role at EDWARD PVT LTD',            'naresh@edward-pvt-ltd.private-sector.dev','edward-pvt-ltd')
ON CONFLICT (id) DO UPDATE SET
	name = EXCLUDED.name,
	description = EXCLUDED.description,
	email = EXCLUDED.email,
	company_id = EXCLUDED.company_id;
