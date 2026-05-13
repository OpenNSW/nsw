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
        '{
            "formId": "npqs-application-form",
            "submission": {
                "serviceId": "npqs",
                "url": "/api/review",
                "request": {
                    "taskCode": "npqs_application_v1"
                }
            },
            "callback": {
                "response": {
                    "display": {
                        "formId": "npqs-application-review-response"
                    },
                    "mapping": {
                        "review_outcome":      "review_outcome",
                        "reference_number":    "reference_number",
                        "sample_required":     "sample_required",
                        "fumigation_required": "fumigation_required"
                    }
                },
                "transition": {
                    "field":   "review_outcome",
                    "default": "OGA_VERIFICATION_APPROVED",
                    "mapping": {
                        "needs_more_info": "OGA_VERIFICATION_FEEDBACK"
                    }
                }
            }
        }',
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
        '{
            "display": {
                "title": {
                    "waiting":   "Waiting for Sample Receipt",
                    "failed":    "Sample Receipt Notification Failed",
                    "completed": "Sample Receipt Confirmed"
                },
                "description": {
                    "waiting":   "Your consignment has been registered. The NPQS facility will confirm receipt of the physical sample before lab testing begins.",
                    "failed":    "The sample receipt notification could not be delivered. Please contact the NPQS helpdesk.",
                    "completed": "The NPQS facility has confirmed receipt of your consignment sample. Lab testing is now in progress."
                }
            },
            "submission": {
                "serviceId": "npqs",
                "url":       "/api/queue",
                "request": {
                    "taskCode": "npqs_sample_wait_v1",
                    "template": {
                        "referenceNumber": "npqs_reference_number"
                    }
                }
            }
        }',
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
        '{
            "display": {
                "title": {
                    "waiting":   "Waiting for Lab Test Results",
                    "failed":    "Lab Result Notification Failed",
                    "completed": "Lab Test Results Received"
                },
                "description": {
                    "waiting":   "Laboratory testing is underway. You will be notified once results are available.",
                    "failed":    "The lab result notification could not be delivered. Please contact the NPQS helpdesk.",
                    "completed": "Laboratory testing is complete. Please check the test outcome below."
                }
            },
            "submission": {
                "serviceId": "npqs",
                "url":       "/api/queue",
                "request": {
                    "taskCode": "npqs_lab_wait_v1",
                    "template": {
                        "referenceNumber": "npqs_reference_number"
                    }
                },
                "response": {
                    "mapping": {
                        "lab_result": "lab_result"
                    }
                }
            }
        }',
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
        '{
            "display": {
                "title": {
                    "waiting":   "Waiting for Fumigation Treatment Completion",
                    "failed":    "Fumigation Notification Failed",
                    "completed": "Fumigation Treatment Completed"
                },
                "description": {
                    "waiting":   "Fumigation treatment is being arranged. You will be notified once it is complete.",
                    "failed":    "The fumigation completion notification could not be delivered. Please contact the NPQS helpdesk.",
                    "completed": "Fumigation treatment has been completed and certified. The process will now continue."
                }
            },
            "submission": {
                "serviceId": "npqs",
                "url":       "/api/queue",
                "request": {
                    "taskCode": "npqs_fumigation_wait_v1",
                    "template": {
                        "referenceNumber": "npqs_reference_number"
                    }
                }
            }
        }',
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
        '{
            "display": {
                "title": {
                    "waiting":   "Awaiting Visual Inspection Decision",
                    "failed":    "Visual Inspection Decision Notification Failed",
                    "completed": "Visual Inspection Decision Received"
                },
                "description": {
                    "waiting":   "An NPQS officer is determining whether a visual inspection of your consignment is required.",
                    "failed":    "The visual inspection decision notification could not be delivered. Please contact the NPQS helpdesk.",
                    "completed": "The NPQS officer has determined whether a visual inspection is required. The process will continue accordingly."
                }
            },
            "submission": {
                "serviceId": "npqs",
                "url":       "/api/queue",
                "request": {
                    "taskCode": "npqs_visual_decision_v1",
                    "template": {
                        "referenceNumber": "npqs_reference_number"
                    }
                },
                "response": {
                    "mapping": {
                        "visual_inspection_required": "visual_inspection_required"
                    }
                }
            }
        }',
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
        '{
            "display": {
                "title": {
                    "waiting":   "Visual Inspection In Progress",
                    "failed":    "Visual Inspection Notification Failed",
                    "completed": "Visual Inspection Complete"
                },
                "description": {
                    "waiting":   "A physical visual inspection of your consignment is being carried out by an NPQS officer.",
                    "failed":    "The visual inspection result notification could not be delivered. Please contact the NPQS helpdesk.",
                    "completed": "The visual inspection of your consignment is complete. Please check the outcome below."
                }
            },
            "submission": {
                "serviceId": "npqs",
                "url":       "/api/queue",
                "request": {
                    "taskCode": "npqs_visual_result_v1",
                    "template": {
                        "referenceNumber": "npqs_reference_number"
                    }
                },
                "response": {
                    "mapping": {
                        "visual_result": "visual_result"
                    }
                }
            }
        }',
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
        '{
            "formId": "npqs-shipping-docs-form",
            "submission": {
                "serviceId": "npqs",
                "url": "/api/review",
                "request": {
                    "taskCode": "npqs_shipping_docs_v1",
                    "template": {
                        "referenceNumber": "npqs_reference_number"
                    }
                }
            },
            "callback": {
                "response": {
                    "display": {
                        "formId": "npqs-shipping-docs-review-response"
                    },
                    "mapping": {
                        "review_outcome": "doc_review_result"
                    }
                },
                "transition": {
                    "field":   "review_outcome",
                    "default": "OGA_VERIFICATION_APPROVED",
                    "mapping": {
                        "needs_more_info": "OGA_VERIFICATION_FEEDBACK"
                    }
                }
            }
        }',
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
        '{
            "currency":    "LKR",
            "ttl":         3600,
            "orgId":       "NPQS",
            "serviceType": "PHYTOSANITARY CERTIFICATE",
            "breakdown": [
                {
                    "description": "Phytosanitary Certificate Processing Fee",
                    "category":    "ADDITION",
                    "type":        "FIXED",
                    "quantity":    "1",
                    "unitPrice":   "1500.00"
                },
                {
                    "description": "VAT",
                    "category":    "ADDITION",
                    "type":        "PERCENTAGE",
                    "value":       "18"
                }
            ]
        }',
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
        '{
            "display": {
                "title": {
                    "waiting":   "Awaiting Certificate Issuance",
                    "failed":    "Certificate Issuance Notification Failed",
                    "completed": "Phytosanitary Certificate Issued"
                },
                "description": {
                    "waiting":   "An NPQS officer is finalising and issuing your phytosanitary certificate. You will be notified once it is ready.",
                    "failed":    "The certificate issuance notification could not be delivered. Please contact the NPQS helpdesk.",
                    "completed": "Your phytosanitary certificate has been issued successfully. The certificate details are shown below."
                }
            },
            "submission": {
                "serviceId": "npqs",
                "url":       "/api/queue",
                "request": {
                    "taskCode": "npqs_certificate_issue_v1",
                    "template": {
                        "referenceNumber": "npqs_reference_number"
                    }
                },
                "response": {
                    "display": {
                        "formId": "npqs-certificate-officer-form"
                    },
                    "mapping": {
                        "certificate_id":  "certificate_id",
                        "certificate_url": "certificate_url"
                    }
                }
            }
        }',
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
        '{
            "display": {
                "title": {
                    "waiting":   "Uploading to IPPC Hub",
                    "failed":    "IPPC Hub Upload Notification Failed",
                    "completed": "IPPC Hub Upload Complete"
                },
                "description": {
                    "waiting":   "Your phytosanitary certificate is being registered with the International Plant Protection Convention (IPPC) hub.",
                    "failed":    "The IPPC upload notification could not be delivered. Please contact the NPQS helpdesk.",
                    "completed": "Your phytosanitary certificate has been successfully registered with the IPPC hub. The process is now complete."
                }
            },
            "submission": {
                "serviceId": "npqs",
                "url":       "/api/queue",
                "request": {
                    "taskCode": "npqs_ippc_upload_v1",
                    "template": {
                        "referenceNumber": "npqs_reference_number",
                        "certificateId":   "npqs_certificate_id",
                        "certificateUrl":  "npqs_certificate_url"
                    }
                }
            }
        }',
        '[]'
    );