package utils

const CustomsWorkflowJSON = `
{
  "workflow_id": "trade-export-v1",
  "name": "General Information & Certificate Approvals",
  "version": 1,
  "edges": [
    { "id": "e_start", "source_id": "node_0_start", "target_id": "node_1_gen_info" },
    { "id": "e_gen_info_to_cusdec", "source_id": "node_1_gen_info", "target_id": "node_2_cusdec" },
    { "id": "e_cusdec_to_payment", "source_id": "node_2_cusdec", "target_id": "node_3_payment" },
    
    { "id": "e_payment_to_split", "source_id": "node_3_payment", "target_id": "gw_4_parallel_split" },
    { "id": "e_split_to_phyto", "source_id": "gw_4_parallel_split", "target_id": "node_5_phyto" },
    { "id": "e_split_to_health", "source_id": "gw_4_parallel_split", "target_id": "node_6_health" },

    { "id": "e_phyto_to_eval", "source_id": "node_5_phyto", "target_id": "gw_7_exclusive_split" },
    { "id": "e_eval_to_manual", "source_id": "gw_7_exclusive_split", "target_id": "node_8_manual_inspect", "condition": "phyto_outcome == 'npqs:phytosanitary:manual_review_required'" },
    { "id": "e_eval_to_phyto_join", "source_id": "gw_7_exclusive_split", "target_id": "gw_9_exclusive_join", "condition": "phyto_outcome == 'npqs:phytosanitary:approved'" },
    { "id": "e_manual_to_phyto_join", "source_id": "node_8_manual_inspect", "target_id": "gw_9_exclusive_join" },

    { "id": "e_phyto_join_to_final", "source_id": "gw_9_exclusive_join", "target_id": "gw_10_parallel_join" },
    { "id": "e_health_to_final", "source_id": "node_6_health", "target_id": "gw_10_parallel_join" },

    { "id": "e_final_join_to_process", "source_id": "gw_10_parallel_join", "target_id": "node_11_final_process" },
    { "id": "e_process_to_end", "source_id": "node_11_final_process", "target_id": "node_12_end" }
  ],
  "nodes": [
    { "id": "node_0_start", "type": "START" },
    { "id": "node_1_gen_info", "type": "TASK", "task_template_id": "c0000003-0003-0003-0003-000000000001" },
    { "id": "node_2_cusdec", "type": "TASK", "task_template_id": "c0000003-0003-0003-0003-000000000002" },
    { "id": "node_3_payment", "type": "TASK", "task_template_id": "c0000003-0003-0003-0003-000000000008" },
    
    { "id": "gw_4_parallel_split", "type": "GATEWAY", "gateway_type": "PARALLEL_SPLIT" },
    
    { "id": "node_5_phyto", "type": "TASK", "task_template_id": "c0000003-0003-0003-0003-000000000003", "output_mapping": { "outcome": "phyto_outcome" } },
    { "id": "node_6_health", "type": "TASK", "task_template_id": "c0000003-0003-0003-0003-000000000004" },
    
    { "id": "gw_7_exclusive_split", "type": "GATEWAY", "gateway_type": "EXCLUSIVE_SPLIT" },
    { "id": "node_8_manual_inspect", "type": "TASK", "task_template_id": "e1a00001-0001-4000-b000-000000000007" },
    { "id": "gw_9_exclusive_join", "type": "GATEWAY", "gateway_type": "EXCLUSIVE_JOIN" },
    
    { "id": "gw_10_parallel_join", "type": "GATEWAY", "gateway_type": "PARALLEL_JOIN" },
    
    { "id": "node_11_final_process", "type": "TASK", "task_template_id": "e1a00001-0001-4000-b000-000000000005" },
    { "id": "node_12_end", "type": "END" }
  ]
}
`
