package utils

const CustomsWorkflowJSON = `
{
  "workflow_id": "customs-export-v1",
  "name": "Customs Export Declaration & Release",
  "version": 1,
  "edges":[
    { "id": "e_customs_start", "source_id": "customs_0_start", "target_id": "customs_1_cusdec_submit" },
    { "id": "e_customs_submit_to_pay", "source_id": "customs_1_cusdec_submit", "target_id": "customs_2_duty_payment" },
    { "id": "e_customs_pay_to_warrant", "source_id": "customs_2_duty_payment", "target_id": "customs_3_warranting_gw" },
    { "id": "e_customs_warrant_lcl", "source_id": "customs_3_warranting_gw", "target_id": "customs_4_lcl_cdn_create", "condition": "consignment_type == 'LCL'" },
    { "id": "e_customs_warrant_fcl", "source_id": "customs_3_warranting_gw", "target_id": "customs_4_fcl_cdn_create", "condition": "consignment_type == 'FCL'" },
    { "id": "e_customs_lcl_ack", "source_id": "customs_4_lcl_cdn_create", "target_id": "customs_5_cdn_ack" },
    { "id": "e_customs_fcl_ack", "source_id": "customs_4_fcl_cdn_create", "target_id": "customs_5_cdn_ack" },
    { "id": "e_customs_ack_bn_create", "source_id": "customs_5_cdn_ack", "target_id": "customs_6_boatnote_create" },
    { "id": "e_customs_bn_create_to_appr", "source_id": "customs_6_boatnote_create", "target_id": "customs_6_boatnote_approve" },
    { "id": "e_customs_bn_done", "source_id": "customs_6_boatnote_approve", "target_id": "customs_7_export_released" }
  ],
  "nodes":[
    { "id": "customs_0_start", "type": "START" },
    { "id": "customs_1_cusdec_submit", "type": "TASK", "task_template_id": "SUBMIT_CUSDEC", "output_mapping": { "consignment_type": "consignment_type" } },
    { "id": "customs_2_duty_payment", "type": "TASK", "task_template_id": "PAY_DUTIES" },
    { "id": "customs_3_warranting_gw", "type": "GATEWAY", "gateway_type": "EXCLUSIVE_SPLIT" },
    { "id": "customs_4_lcl_cdn_create", "type": "TASK", "task_template_id": "CREATE_LCL_CDN" },
    { "id": "customs_4_fcl_cdn_create", "type": "TASK", "task_template_id": "CREATE_FCL_CDN" },
    { "id": "customs_5_cdn_ack", "type": "TASK", "task_template_id": "ACK_CDNS" },
    { "id": "customs_6_boatnote_create", "type": "TASK", "task_template_id": "CREATE_BOAT_NOTE" },
    { "id": "customs_6_boatnote_approve", "type": "TASK", "task_template_id": "APPROVE_BOAT_NOTE" },
    { "id": "customs_7_export_released", "type": "END" }
  ]
}`
