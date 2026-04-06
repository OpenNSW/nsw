INSERT INTO workflow_template_v2 (id, name, version, workflow_definition)
VALUES
(
    'fcau-v1',
    'FCAU Workflow',
    '1',
    '{
        "id": "fcau-v1",
        "name": "FCAU Workflow",
        "version": 1,
        "nodes": [
            { "id": "start_1", "type": "START" },
            { "id": "node_1:application_submission", "type": "TASK", "task_template_id": "fcau:application_submission", "output_mapping": { } },
            { "id": "node_2:wait_sample_drop", "type": "TASK", "task_template_id": "fcau:sample_drop", "input_mapping": { }, "output_mapping": { } },
            { "id": "node_3:wait_testing_requirement", "type": "TASK", "task_template_id": "fcau:testing_requirement", "input_mapping": { }, "output_mapping": { "requires_lab_test": "fcau:requires_lab_test" } },

            { "id": "gw_1:requires_lab_test", "type": "GATEWAY", "gateway_type": "EXCLUSIVE_SPLIT" },

            { "id": "node_4:lab_payment_upload", "type": "TASK", "task_template_id": "fcau:lab_payment_upload", "input_mapping": { }, "output_mapping": { } },
            { "id": "node_5:lab_results_review", "type": "TASK", "task_template_id": "fcau:lab_results_review", "input_mapping": { }, "output_mapping": { "lab_decision": "fcau:lab_decision" } },

            { "id": "gw_2:lab_pass_or_fail", "type": "GATEWAY", "gateway_type": "EXCLUSIVE_SPLIT"},
            { "id": "end_1:lab_decision_failed", "type": "END" },

            { "id": "gw_3:proceed_to_payment", "type": "GATEWAY", "gateway_type": "EXCLUSIVE_JOIN" },

            { "id": "node_6:payment", "type": "TASK", "task_template_id": "fcau:payment", "input_mapping": { }, "output_mapping": { } },
            { "id": "node_7:certificate_issue", "type": "TASK", "task_template_id": "fcau:certificate_issue", "input_mapping": { }, "output_mapping": { } },

            { "id": "end_2:fcau_process_complete", "type": "END" }
           
        ],
        "edges": [
            { "id": "e_1:start_to_app_sub", "source_id": "start_1", "target_id": "node_1:application_submission" },
            { "id": "e_2:app_sub_to_wait_sample", "source_id": "node_1:application_submission", "target_id": "node_2:wait_sample_drop" },
            { "id": "e_3:wait_sample_to_wait_testing_req", "source_id": "node_2:wait_sample_drop", "target_id": "node_3:wait_testing_requirement" },
            { "id": "e_4:wait_testing_to_gateway", "source_id": "node_3:wait_testing_requirement", "target_id": "gw_1:requires_lab_test" },

            { "id": "e_5:gateway_to_lab_payment", "source_id": "gw_1:requires_lab_test", "target_id": "node_4:lab_payment_upload", "condition": "fcau:requires_lab_test == true" },
            { "id": "e_6:gateway_to_join", "source_id": "gw_1:requires_lab_test", "target_id": "gw_3:proceed_to_payment", "condition": "fcau:requires_lab_test == false" },

            { "id": "e_7:lab_payment_to_results_review", "source_id": "node_4:lab_payment_upload", "target_id": "node_5:lab_results_review" },
            { "id": "e_8:results_review_to_gateway", "source_id": "node_5:lab_results_review", "target_id": "gw_2:lab_pass_or_fail" },
            { "id": "e_9:gateway_to_failed_end", "source_id": "gw_2:lab_pass_or_fail", "target_id": "end_1:lab_decision_failed", 	"condition": "fcau:lab_decision == ''failed''" },
            { "id": "e_10:gateway_to_join", "source_id": "gw_2:lab_pass_or_fail", "target_id": "gw_3:proceed_to_payment", "condition": "fcau:lab_decision == ''passed''" },

            { "id": "e_11:join_to_payment", "source_id": "gw_3:proceed_to_payment", "target_id": "node_6:payment" },
            { "id": "e_12:payment_to_cert_issue", "source_id": "node_6:payment", "target_id": "node_7:certificate_issue" },
            { "id": "e_13:cert_issue_to_end", "source_id": "node_7:certificate_issue", "target_id": "end_2:fcau_process_complete" }           
        ]
    }'::jsonb
);


-- Purpose: Seed workflow templates and mappings for the FCAU process.
INSERT INTO hs_codes (id, hs_code, description, category)
VALUES
    (
        'fcau-hs-code-0001',
        'fcau-process',
        'HS code representing the FCAU process for testing.',
        'FCAU'
    );

INSERT INTO workflow_template_maps_v2 (id, hs_code_id, consignment_flow, workflow_template_id)
VALUES
    -- Mapping for FCAU process
    (
        'fcau-wf-map-0001',
        'fcau-hs-code-0001',
        'EXPORT',
        'fcau-v1'
    );