-- Migration: 002_insert_seed_hscodes.down.sql
-- Description: Roll back HS code seed data.

DELETE FROM hs_codes 
WHERE id IN (
    '90b06747-cfa7-486b-a084-eaa1fc95595e',
    '3699f18c-832a-4026-ac31-3697c3a5235d',
    '1589b5b1-2db3-44ef-80c1-16151bb8d5b0',
    '6aa146ba-dd72-4e5e-ae27-a1cb5d69caa5',
    '2e173ef8-840b-4cc5-a667-03e1d80e04b9',
    '851f0de7-0693-4cc1-9d92-19c39072bb53',
    '51e802c1-b57e-45ac-b563-1ae0fad06db5',
    'cb34d1ac-c48f-4370-8260-a6585009ff7e',
    '36a58d44-8ff6-4bea-8c9b-3db84bb5a083',
    '8a0783e4-82e6-488e-b96e-6140a8912f39',
    '4bdfb1f0-2b71-4ddc-8b99-f31c3d7660bc',
    'b9e48207-2573-4c9b-89f6-06d4c22422be',
    '6b567998-4a57-4132-a595-577493aefb3f',
    '653c4c8f-8c39-4aee-86f5-7f3926d0d4c2',
    'bfa92119-64d3-41f4-b21c-fd0e2eb2966b',
    '5e0f2a51-8a1e-4d7d-a00b-4565e47535d2',
    '4f4fac26-bf5c-42b0-9058-b17828dcba31',
    '1390c617-43d4-4eee-8fff-b9f10d038981',
    'fd5a0de1-c547-4420-94b9-942a8349a463',
    '7884654e-90e0-4b7c-a963-cf6d2b5d1c16',
    '4ba1fd6b-f42f-438f-ab9f-0ee0054ee33c'
);