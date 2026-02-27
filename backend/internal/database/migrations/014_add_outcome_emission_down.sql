UPDATE workflow_node_templates
SET config = '{
  "agency": "NPQS",
  "formId": "22222222-2222-2222-2222-222222222222",
  "service": "plant-quarantine-phytosanitary",
  "callback": {
    "transition": {
      "field": "decision",
      "mapping": {
        "APPROVED": "OGA_VERIFICATION_APPROVED",
        "MANUAL_REVIEW": "OGA_VERIFICATION_APPROVED"
      },
      "default": "OGA_VERIFICATION_REJECTED"
    },
    "response": {
      "display": {
        "formId": "d0c3b860-635b-4124-8081-d3f421e429cb"
      },
      "mapping": {
        "reviewedAt": "gi:phytosanitary:meta:reviewedAt",
        "reviewerNotes": "gi:phytosanitary:meta:reviewNotes"
      }
    }
  },
  "submission": {
    "url": "http://localhost:8081/api/oga/inject",
    "request": {
      "meta": {
        "type": "consignment",
        "verificationId": "moa:npqs:phytosanitary:001"
      }
    }
  },
}'
WHERE id = 'c0000003-0003-0003-0003-000000000003';
