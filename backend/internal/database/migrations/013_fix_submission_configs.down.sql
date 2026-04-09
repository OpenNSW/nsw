-- Migration: 013_fix_submission_configs.down.sql
-- Down migration would theoretically revert the serviceId pattern, but since this is a 
-- data fixup, reverting is complex. Leaving empty as the up migration is corrective.
SELECT 1;
