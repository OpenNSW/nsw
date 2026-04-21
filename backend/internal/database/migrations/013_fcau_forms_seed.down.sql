-- Migration: 012_fcau_forms_seed.down.sql
-- Description: Roll back form template seed data.

DELETE FROM forms 
WHERE id IN (
    'fcau-application-form',
    'fcau-application-review-response',
    'fcau-sample-drop-ack-response',
    'fcau-testing-requirement-response',
    'fcau-lab-payment-form',
    'fcau-lab-payment-review-response',
    'fcau-lab-results-review-response',
    'fcau-certificate-issue-response'
);
