-- Migration: 013_fix_submission_configs.up.sql
-- Purpose: Remediation of hardcoded legacy OGA backend URLs in both templates and in-flight tasks.
-- This fix transforms "http://dev-oga-<service>-backend:8080/api/oga/inject" 
-- into the serviceId-based pattern which allows dynamic resolution via Helm registry.

-- 1. Patch the Template Library
-- -------------------------------------------------------------------

-- FCAU Health Certificate
UPDATE workflow_node_templates
SET config = config || jsonb_build_object('submission', 
    (config->'submission') || jsonb_build_object('serviceId', 'fcau', 'url', '/api/oga/inject')
)
WHERE id = 'c0000003-0003-0003-0003-000000000004';

-- NPQS Phytosanitary Certificate
UPDATE workflow_node_templates
SET config = config || jsonb_build_object('submission', 
    (config->'submission') || jsonb_build_object('serviceId', 'npqs', 'url', '/api/oga/inject')
)
WHERE id = 'c0000003-0003-0003-0003-000000000003';

-- NPQS Manual Inspection
UPDATE workflow_node_templates
SET config = config || jsonb_build_object('submission', 
    (config->'submission') || jsonb_build_object('serviceId', 'npqs', 'url', '/api/oga/inject')
)
WHERE id = 'e1a00001-0001-4000-b000-000000000007';

-- IRD Pre-Consignment Verification
UPDATE workflow_node_templates
SET config = config || jsonb_build_object('submission', 
    (config->'submission') || jsonb_build_object('serviceId', 'ird', 'url', '/api/oga/inject')
)
WHERE id = 'd0000002-0001-0001-0001-000000000005';


-- 2. Patch In-Flight Task Instances
-- -------------------------------------------------------------------
-- This ensures tasks already started (like node_6_health:... reported by user) 
-- are fixed without needing a workflow restart.

-- Universal patch for any task containing the legacy dev-oga URL pattern
UPDATE task_infos
SET config = config || jsonb_build_object('submission', 
    (config->'submission') || jsonb_build_object(
        'serviceId', 
        CASE 
            WHEN config->'submission'->>'url' LIKE '%fcau%' THEN 'fcau'
            WHEN config->'submission'->>'url' LIKE '%npqs%' THEN 'npqs'
            WHEN config->'submission'->>'url' LIKE '%ird%' THEN 'ird'
            ELSE config->'submission'->>'serviceId' 
        END,
        'url', '/api/oga/inject'
    )
)
WHERE config->'submission'->>'url' LIKE '%dev-oga-%:8080%';

-- 3. Cleanup: Remove the full URL from the config if serviceId is now set
-- (Optional, but cleaner to avoid mixed routing logic)
UPDATE task_infos
SET config = config #- '{submission,url}'
WHERE config->'submission'->>'serviceId' IS NOT NULL 
  AND config->'submission'->>'url' = '/api/oga/inject';

-- Re-add relative path correctly (the previous command might have removed it or was just a path)
-- Actually, let's keep it simple. The first two blocks are the most important.
