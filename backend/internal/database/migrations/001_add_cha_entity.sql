-- Migration: 001_add_cha_entity.sql

-- 1. Create the CHA agency table (Customs House Agent)
CREATE TABLE customs_house_agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 2. Link CHAs to Consignments (nullable for backward compatibility)
ALTER TABLE consignments ADD COLUMN cha_id UUID REFERENCES customs_house_agents(id);
CREATE INDEX idx_consignments_cha_id ON consignments (cha_id);

-- 3. Seed Major Service Providers
INSERT INTO customs_house_agents (name, description) VALUES 
('Spectra Logistics', 'Specializes in end-to-end logistics and customs brokerage'),
('Aitken Spence Freight', 'Handles comprehensive customs clearance'),
('Advantis Projects', 'Offers experienced clearance services'),
('Yusen Logistics', 'Provides Customs House Brokerage services'),
('Malship Group', 'Handles various cargo, including bulk and containerized');
