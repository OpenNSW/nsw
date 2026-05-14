-- ============================================================================
-- Migration: 017_npqs_workflow_nodes_seed.up.sql
-- Purpose: Seed WorkflowNodeTemplates for the NPQS phytosanitary workflow.
--
-- Level 1 (Application & Review):
--   npqs:application_submission   SIMPLE_FORM  — trader applies, officer reviews
--
-- Level 2 (Testing & Compliance):
--   npqs:sample_wait              WAIT_FOR_EVENT — wait for sample receipt
--   npqs:lab_wait                 WAIT_FOR_EVENT — wait for lab result
--   npqs:fumigation_wait          WAIT_FOR_EVENT — wait for fumigation completion
--   npqs:visual_decision_wait     WAIT_FOR_EVENT — officer decides visual inspection need
--   npqs:visual_result_wait       WAIT_FOR_EVENT — wait for visual inspection result
--   npqs:shipping_docs_submission SIMPLE_FORM  — trader uploads docs, officer reviews
--
-- Level 3 (Finalization):
--   npqs:payment                  PAYMENT        — process phytosanitary certificate fee
--   npqs:certificate_issue        WAIT_FOR_EVENT — officer issues certificate via callback
--   npqs:ippc_upload              WAIT_FOR_EVENT — notify IPPC hub and wait for confirmation
-- ============================================================================

INSERT INTO workflow_node_templates (id, name, description, type, config, depends_on)
VALUES

    -- =========================================================================
    -- Level 1: Application Submission & Officer Review (SIMPLE_FORM with callback)
    -- =========================================================================
    (
        'npqs-apply-phyto-cert-flow',
        'NPQS Phytosanitary Application',
        'Trader submits phytosanitary export application. NPQS officer reviews and provides a decision (approve / reject / needs more info). The approval also sets sample_required and fumigation_required flags.',
        'SIMPLE_FORM',
        '{}'::jsonb,
        '[]'
    ),

    -- =========================================================================
    -- Level 2: Sample Receipt Wait (WAIT_FOR_EVENT)
    -- =========================================================================
    (
        'npqs-wait-sample-received-flow',
        'Wait for Sample Receipt',
        'Waits for the NPQS facility to confirm physical receipt of the consignment sample. Notifies the NPQS queue service with the reference number.',
        'WAIT_FOR_EVENT',
        '{}'::jsonb,
        '[]'
    ),

    -- =========================================================================
    -- Level 2: Lab Result Wait (WAIT_FOR_EVENT)
    -- Outputs: lab_result — mapped by workflow to npqs_lab_result
    -- =========================================================================
    (
        'npqs-wait-lab-result-flow',
        'Wait for Lab Result',
        'Waits for the NPQS lab to return a pass/fail result. On callback the lab_result field is extracted and propagated to the workflow context.',
        'WAIT_FOR_EVENT',
        '{}'::jsonb,
        '[]'
    ),

    -- =========================================================================
    -- Level 2: Fumigation Wait (WAIT_FOR_EVENT)
    -- =========================================================================
    (
        'npqs-wait-fumigation-flow',
        'Wait for Fumigation Completion',
        'Waits for the fumigation treatment to be completed and certified before proceeding to visual inspection.',
        'WAIT_FOR_EVENT',
        '{}'::jsonb,
        '[]'
    ),

    -- =========================================================================
    -- Level 2: Visual Inspection Decision (WAIT_FOR_EVENT)
    -- Outputs: visual_inspection_required — mapped to npqs_visual_inspection_required
    -- =========================================================================
    (
        'npqs-wait-visual-decision-flow',
        'Visual Inspection Requirement Check',
        'NPQS officer determines whether a visual inspection is required for this consignment. Result is fed back via callback.',
        'WAIT_FOR_EVENT',
        '{}'::jsonb,
        '[]'
    ),

    -- =========================================================================
    -- Level 2: Visual Inspection Result (WAIT_FOR_EVENT)
    -- Outputs: visual_result — mapped to npqs_visual_result
    -- =========================================================================
    (
        'npqs-visual-inspection-result-flow',
        'Visual Inspection Result',
        'Waits for the visual inspection of the consignment to be completed. Returns a pass/fail result.',
        'WAIT_FOR_EVENT',
        '{}'::jsonb,
        '[]'
    ),

    -- =========================================================================
    -- Level 2/3 boundary: Shipping Documents Submission & Review (SIMPLE_FORM)
    -- Outputs: doc_review_result — mapped to npqs_doc_review_result
    -- =========================================================================
    (
        'npqs-submit-shipping-docs-flow',
        'Shipping Documents Submission & Review',
        'Trader uploads required shipping documents (Bill of Lading, Packing List, Commercial Invoice). NPQS officer reviews and approves or requests corrections.',
        'SIMPLE_FORM',
        '{}'::jsonb,
        '[]'
    ),

    -- =========================================================================
    -- Level 3: Payment (PAYMENT)
    -- Outputs: payment_status, payment_reference_number
    -- =========================================================================
    (
        'npqs-pay-certificate-fee-flow',
        'Phytosanitary Certificate Fee Payment',
        'Processes the NPQS phytosanitary certificate issuance fee. On success emits payment_status=success and the payment reference number.',
        'PAYMENT',
        '{}'::jsonb,
        '[]'
    ),

    -- =========================================================================
    -- Level 3: Certificate Issuance (WAIT_FOR_EVENT)
    -- Outputs: certificate_id, certificate_url
    -- =========================================================================
    (
        'npqs-issue-certificate-flow',
        'Phytosanitary Certificate Issuance',
        'NPQS officer issues the phytosanitary certificate and provides the certificate ID and URL via callback.',
        'WAIT_FOR_EVENT',
        '{}'::jsonb,
        '[]'
    ),

    -- =========================================================================
    -- Level 3: IPPC Hub Upload (WAIT_FOR_EVENT acting as fire-and-confirm)
    -- =========================================================================
    (
        'npqs-upload-ippc-flow',
        'IPPC Hub Registration Upload',
        'Notifies the NPQS service to upload the issued certificate to the IPPC hub. Waits for upload confirmation.',
        'WAIT_FOR_EVENT',
        '{}'::jsonb,
        '[]'
    );