INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES
('b1d0a101-0001-4000-8000-100000000001', 
'General Info', 
'General Info', 
'{
  "type": "object",
  "title": "Shipment Initialization",
  "properties": {
    "exporterName": {
      "type": "string",
      "title": "Exporter Name & Address",
      "x-globalContext": { "writeTo": "global_exporter_name" },
      "example": "Ceylon Coconut Exports PLC, No 45, Colombo 01"
    },
    "exporterTin": {
      "type": "string",
      "title": "Exporter TIN",
      "x-globalContext": { "writeTo": "global_exporter_tin" },
      "example": "TIN-99283341"
    },
    "consigneeName": {
      "type": "string",
      "title": "Consignee Name & Address",
      "x-globalContext": { "writeTo": "global_consignee_name" },
      "example": "Global Foods GMBH, Berlin, Germany"
    },
    "declarantName": {
      "type": "string",
      "title": "Declarant/Agent Name",
      "x-globalContext": { "writeTo": "global_declarant_name" },
      "example": "Logistics Pro Clearing (Pvt) Ltd"
    }
  }
}',
'{
  "type": "VerticalLayout",
  "elements": [
    {
      "type": "Group",
      "label": "Exporter & Tax Information",
      "elements": [
        {
          "type": "HorizontalLayout",
          "elements": [
            {
              "type": "Control",
              "scope": "#/properties/exporterName",
              "label": "Registered Company Name"
            },
            {
              "type": "Control",
              "scope": "#/properties/exporterTin",
              "label": "Tax ID (TIN)"
            }
          ]
        }
      ]
    },
    {
      "type": "Group",
      "label": "Consignee (Buyer) Details",
      "elements": [
        {
          "type": "Control",
          "scope": "#/properties/consigneeName",
          "label": "Buyer Name & Destination Address"
        }
      ]
    },
    {
      "type": "Group",
      "label": "Authorized Representative",
      "elements": [
        {
          "type": "Control",
          "scope": "#/properties/declarantName",
          "label": "Clearing Agent / Declarant"
        }
      ]
    }
  ]
}',
'1.0',
true);

INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES
('b1d0a101-0001-4000-8000-200000000001', 
'7ai: Custom Declaration', 
'Comprehensive Customs Declaration Form with Global Context', 
'{
  "type": "object",
  "title": "7ai: Customs Declaration",
  "properties": {
    "exporterDetails": {
      "type": "string",
      "title": "1. Exporter",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_exporter_name" }
    },
    "consigneeDetails": {
      "type": "string",
      "title": "8. Consignee",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_consignee_name" }
    },
    "declarantDetails": {
      "type": "string",
      "title": "14. Declarant / Representative",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_declarant_name" }
    },
    "declarationType": {
      "type": "string",
      "title": "1. Declaration Type",
      "enum": ["EX1 - Export to Third Country", "EX2 - Temporary Export", "EX3 - Re-export"],
      "example": "EX1 - Export to Third Country"
    },
    "countryDestination": {
      "type": "string",
      "title": "17. Country of Destination",
      "x-globalcontext": { "writeTo": "global_destination_country" },
      "example": "Germany (DE)"
    },
    "totalItems": {
      "type": "integer",
      "title": "5. Total Items",
      "example": 1
    },
    "packageSummary": {
      "type": "string",
      "title": "31. Packages & Description",
      "example": "450 Bags of Desiccated Coconut (Fine Grade)"
    },
    "totalPackages": {
      "type": "integer",
      "title": "6. Total Packages",
      "x-globalcontext": { "writeTo": "global_package_count" },
      "example": 450
    },
    "totalGrossMass": {
      "type": "number",
      "title": "35. Total Gross Mass (Kg)",
      "x-globalcontext": { "writeTo": "global_total_weight" },
      "example": 11250.00
    },
    "totalInvoicedValue": {
      "type": "number",
      "title": "22. Total Invoice Value (USD)",
      "x-globalcontext": { "writeTo": "global_invoice_value" },
      "example": 24500.00
    }
  }
}',
'{
  "type": "VerticalLayout",
  "elements": [
    {
      "type": "Group",
      "label": "Parties to the Transaction",
      "elements": [
        { "type": "Control", "scope": "#/properties/exporterDetails" },
        { "type": "Control", "scope": "#/properties/consigneeDetails" },
        { "type": "Control", "scope": "#/properties/declarantDetails" }
      ]
    },
    {
      "type": "Group",
      "label": "General Information",
      "elements": [
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/declarationType" },
            { "type": "Control", "scope": "#/properties/countryDestination" }
          ]
        }
      ]
    },
    {
      "type": "Group",
      "label": "Item Summary",
      "elements": [
        { "type": "Control", "scope": "#/properties/packageSummary" },
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/totalItems" },
            { "type": "Control", "scope": "#/properties/totalPackages" },
            { "type": "Control", "scope": "#/properties/totalGrossMass" }
          ]
        },
        { "type": "Control", "scope": "#/properties/totalInvoicedValue" }
      ]
    }
  ]
}',
'1.0',
true);

INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES
('b1d0a101-0001-4000-8000-200000000002', 
'7a-Fin: Financial Settlement', 
'Assessment Notice and Payment Confirmation for Customs Fees', 
'{
  "type": "object",
  "title": "7a-Fin: Financial Settlement",
  "properties": {
    "cusdecRef": {
      "type": "string",
      "title": "Referenced Cusdec No",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_cusdec_no" }
    },
    "invoiceValue": {
      "type": "number",
      "title": "Declared Invoice Value (USD)",
      "x-globalcontext": { "readFrom": "global_invoice_value" }
    },
    "assessmentNoticeNo": {
      "type": "string",
      "title": "Assessment Notice Number",
      "x-globalcontext": { "writeTo": "global_assessment_no" },
      "example": "ASMT/2026/00912",
      "default": "ASMT/2026/00912"
    },
    "cessAmount": {
      "type": "number",
      "title": "Cess Amount (LKR)",
      "readOnly": true,
      "example": 12500.00,
      "default": 12500.00
    },
    "exportLevy": {
      "type": "number",
      "title": "Export Levy (LKR)",
      "readOnly": true,
      "example": 2500.00,
      "default": 2500.00
    },
    "totalPayable": {
      "type": "number",
      "title": "Total Amount Payable (LKR)",
      "readOnly": true,
      "example": 15000.00,
      "default": 15000.00
    },
    "paymentMethod": {
      "type": "string",
      "title": "Payment Method",
      "enum": ["Bank Transfer", "Online Credit Card", "Direct Debit"],
      "example": "Bank Transfer",
      "default": "Bank Transfer"
    },
    "ATTACHMENT_paymentReceipt": {
      "type": "string",
      "title": "ATTACHMENT: Scanned Copy of Payment Receipt",
      "example": "receipt_C12994.pdf"
    }
  }
}',
'{
  "type": "VerticalLayout",
  "elements": [
    {
      "type": "Group",
      "label": "Assessment Reference",
      "elements": [
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/cusdecRef" },
            { "type": "Control", "scope": "#/properties/assessmentNoticeNo" }
          ]
        },
        { "type": "Control", "scope": "#/properties/invoiceValue" }
      ]
    },
    {
      "type": "Group",
      "label": "Tax & Fee Breakdown",
      "elements": [
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/cessAmount" },
            { "type": "Control", "scope": "#/properties/exportLevy" }
          ]
        },
        { "type": "Control", "scope": "#/properties/totalPayable" }
      ]
    },
    {
      "type": "Group",
      "label": "Payment Confirmation",
      "elements": [
        { "type": "Control", "scope": "#/properties/paymentMethod" },
        { "type": "Control", "scope": "#/properties/ATTACHMENT_paymentReceipt" }
      ]
    }
  ]
}',
'1.0',
true);

INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES
('b1d0a101-0001-4000-8000-200000000004', 
'5b: CDA Quality Clearance', 
'Application for Quality Certificate and CDA Review based on Customs data', 
'{
  "type": "object",
  "title": "5b: CDA Quality Clearance",
  "properties": {
    "exporterDetails": {
      "type": "string",
      "title": "Exporter Name",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_exporter_name" }
    },
    "assessmentRef": {
      "type": "string",
      "title": "Linked Assessment Notice No",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_assessment_no" }
    },
    "declaredWeight": {
      "type": "number",
      "title": "Declared Gross Weight (Kg)",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_total_weight" }
    },
    "dcGrade": {
      "type": "string",
      "title": "Desiccated Coconut Grade",
      "enum": ["Fine", "Medium", "Chips", "Flakes"],
      "default": "Fine"
    },
    "millNumber": {
      "type": "string",
      "title": "Source Mill Number",
      "default": "CDA/ML/4492"
    },
    "productionDate": {
      "type": "string",
      "format": "date",
      "title": "Date of Production",
      "default": "2026-02-10"
    },
    "lotNumber": {
      "type": "string",
      "title": "Batch / Lot Number",
      "default": "LOT-2026-001"
    },
    "ATTACHMENT_productionDetail": {
      "type": "string",
      "title": "ATTACHMENT: Production Details (Bags/Weights Breakdown)",
      "default": "prod_details_batch01.pdf"
    }
  }
}',
'{
  "type": "VerticalLayout",
  "elements": [
    {
      "type": "Group",
      "label": "Consignment Verification",
      "elements": [
        { "type": "Control", "scope": "#/properties/exporterDetails" },
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/assessmentRef" },
            { "type": "Control", "scope": "#/properties/declaredWeight" }
          ]
        }
      ]
    },
    {
      "type": "Group",
      "label": "CDA Quality Specifics",
      "elements": [
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/dcGrade" },
            { "type": "Control", "scope": "#/properties/millNumber" }
          ]
        },
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/productionDate" },
            { "type": "Control", "scope": "#/properties/lotNumber" }
          ]
        },
        { "type": "Control", "scope": "#/properties/ATTACHMENT_productionDetail" }
      ]
    }
  ]
}',
'1.0',
true);

INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES
('b1d0a101-0001-4000-8000-200000000008', 
'8ai: Phyto Application', 
'Application for Phytosanitary Certificate - Department of Agriculture', 
'{
  "type": "object",
  "title": "8ai: Phyto Application",
  "properties": {
    "exporterName": {
      "type": "string",
      "title": "Exporter Details",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_exporter_name" }
    },
    "consigneeName": {
      "type": "string",
      "title": "Consignee Details",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_consignee_name" }
    },
    "cusdecRef": {
      "type": "string",
      "title": "Cusdec Number",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_cusdec_no" }
    },
    "botanicalName": {
      "type": "string",
      "title": "Botanical Name of Plants/Products",
      "default": "Cocos nucifera"
    },
    "descriptionOfGoods": {
      "type": "string",
      "title": "Description of Goods",
      "x-globalcontext": { "readFrom": "global_package_summary" },
      "readOnly": true
    },
    "placeOfOrigin": {
      "type": "string",
      "title": "Place of Origin",
      "default": "Sri Lanka"
    },
    "declaredMeansOfTransport": {
      "type": "string",
      "title": "Means of Transport",
      "default": "Sea Freight"
    },
    "pointOfEntry": {
      "type": "string",
      "title": "Declaring Point of Entry",
      "x-globalcontext": { "readFrom": "global_destination_country" },
      "readOnly": true
    },
    "distinguishingMarks": {
      "type": "string",
      "title": "Distinguishing Marks (Container/Seal Nos)",
      "default": "CONU-1234567 / SL-00921"
    },
    "ATTACHMENT_phytoRequestLetter": {
      "type": "string",
      "title": "ATTACHMENT: Request Letter for Inspection",
      "default": "phyto_request_v1.pdf"
    }
  }
}',
'{
  "type": "VerticalLayout",
  "elements": [
    {
      "type": "Group",
      "label": "Consignment Linkage",
      "elements": [
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/cusdecRef" },
            { "type": "Control", "scope": "#/properties/exporterName" }
          ]
        },
        { "type": "Control", "scope": "#/properties/consigneeName" }
      ]
    },
    {
      "type": "Group",
      "label": "Botanical & Transport Details",
      "elements": [
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/botanicalName" },
            { "type": "Control", "scope": "#/properties/placeOfOrigin" }
          ]
        },
        { "type": "Control", "scope": "#/properties/descriptionOfGoods" },
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/declaredMeansOfTransport" },
            { "type": "Control", "scope": "#/properties/pointOfEntry" }
          ]
        },
        { "type": "Control", "scope": "#/properties/distinguishingMarks" }
      ]
    },
    {
      "type": "Group",
      "label": "Supporting Documents",
      "elements": [
        { "type": "Control", "scope": "#/properties/ATTACHMENT_phytoRequestLetter" }
      ]
    }
  ]
}',
'1.0',
true);

INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES
('b1d0a101-0001-4000-8000-200000000010', 
'9-App: Health Application', 
'Application for Health Certificate - Ministry of Health (FCAU)', 
'{
  "type": "object",
  "title": "9-App: Health Application",
  "properties": {
    "exporterName": {
      "type": "string",
      "title": "Exporter Name",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_exporter_name" }
    },
    "consigneeName": {
      "type": "string",
      "title": "Consignee Name",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_consignee_name" }
    },
    "cusdecRef": {
      "type": "string",
      "title": "Cusdec Reference",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_cusdec_no" }
    },
    "natureOfProduct": {
      "type": "string",
      "title": "Nature of Product",
      "x-globalcontext": { "readFrom": "global_package_summary" },
      "readOnly": true
    },
    "batchNumbers": {
      "type": "string",
      "title": "Batch / Code Numbers",
      "default": "B-992/2026"
    },
    "sampleSubmissionDate": {
      "type": "string",
      "format": "date",
      "title": "Date of Sample Submission",
      "default": "2026-02-12"
    },
    "labReference": {
      "type": "string",
      "title": "Laboratory Reference Number (if any)",
      "default": "LAB-H-0042"
    },
    "storageConditions": {
      "type": "string",
      "title": "Storage Conditions During Transport",
      "default": "Dry and Cool Environment"
    },
    "ATTACHMENT_healthApplicationForm": {
      "type": "string",
      "title": "ATTACHMENT: Completed FCAU Application Form",
      "default": "fcau_app_v2.pdf"
    }
  }
}',
'{
  "type": "VerticalLayout",
  "elements": [
    {
      "type": "Group",
      "label": "Consignment Context",
      "elements": [
        { "type": "Control", "scope": "#/properties/cusdecRef" },
        { "type": "Control", "scope": "#/properties/exporterName" },
        { "type": "Control", "scope": "#/properties/consigneeName" }
      ]
    },
    {
      "type": "Group",
      "label": "Health & Safety Details",
      "elements": [
        { "type": "Control", "scope": "#/properties/natureOfProduct" },
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/batchNumbers" },
            { "type": "Control", "scope": "#/properties/sampleSubmissionDate" }
          ]
        },
        { "type": "Control", "scope": "#/properties/labReference" },
        { "type": "Control", "scope": "#/properties/storageConditions" }
      ]
    },
    {
      "type": "Group",
      "label": "FCAU Documentation",
      "elements": [
        { "type": "Control", "scope": "#/properties/ATTACHMENT_healthApplicationForm" }
      ]
    }
  ]
}',
'1.0',
true);

INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES
('b1d0a101-0001-4000-8000-200000000007', 
'7bi: Warranting', 
'System-generated Warranting status based on aggregated OGA and Financial approvals', 
'{
  "type": "object",
  "title": "7bi: Customs Warranting Status",
  "properties": {
    "cusdecRef": {
      "type": "string",
      "title": "Cusdec Number",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_cusdec_no" }
    },
    "paymentVerification": {
      "type": "string",
      "title": "Payment Status (Customs/Cess)",
      "readOnly": true,
      "default": "Verified & Settled",
      "x-globalcontext": { "readFrom": "global_assessment_no" }
    },
    "cdaApprovalStatus": {
      "type": "string",
      "title": "CDA Regulatory Status",
      "readOnly": true,
      "default": "Quality Certificate Issued (Tick Received)"
    },
    "warrantNumber": {
      "type": "string",
      "title": "Official Warrant Number",
      "x-globalcontext": { "writeTo": "global_warrant_no" },
      "default": "W-2026-99102-X"
    },
    "warrantDate": {
      "type": "string",
      "title": "Warranting Timestamp",
      "readOnly": true,
      "default": "2026-02-12 14:30:00"
    },
    "systemRemark": {
      "type": "string",
      "title": "System Remarks",
      "readOnly": true,
      "default": "Consignment is authorized for export movement."
    }
  }
}',
'{
  "type": "VerticalLayout",
  "elements": [
    {
      "type": "Group",
      "label": "Consignment Summary",
      "elements": [
        { "type": "Control", "scope": "#/properties/cusdecRef" },
        { "type": "Control", "scope": "#/properties/paymentVerification" },
        { "type": "Control", "scope": "#/properties/cdaApprovalStatus" }
      ]
    },
    {
      "type": "Group",
      "label": "Warranting Details",
      "elements": [
        { "type": "Control", "scope": "#/properties/warrantNumber" },
        { "type": "Control", "scope": "#/properties/warrantDate" },
        { "type": "Control", "scope": "#/properties/systemRemark" }
      ]
    }
  ]
}',
'1.0',
true);

INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES
('b1d0a101-0001-4000-8000-200000000021', 
'7bii: Cargo Selectivity', 
'Automated Risk Assessment and Lane Assignment', 
'{
  "type": "object",
  "title": "7bii: Cargo Selectivity Result",
  "properties": {
    "warrantRef": {
      "type": "string",
      "title": "Reference Warrant No",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_warrant_no" }
    },
    "riskLevel": {
      "type": "string",
      "title": "Assigned Channel (Lane)",
      "enum": ["GREEN - Document Release", "YELLOW - Document Check", "RED - Physical Examination"],
      "default": "GREEN - Document Release",
      "x-globalcontext": { "writeTo": "global_assigned_lane" }
    },
    "examinationRequired": {
      "type": "boolean",
      "title": "Physical Examination Required?",
      "default": false,
      "x-globalcontext": { "writeTo": "global_exam_required" }
    },
    "selectivityTimestamp": {
      "type": "string",
      "title": "Selectivity Run Time",
      "readOnly": true,
      "default": "2026-02-12 14:45:10"
    },
    "instructions": {
      "type": "string",
      "title": "Customs Instructions",
      "readOnly": true,
      "default": "Proceed to Export Facilitation Center (EFC) for gating. No physical examination required for this consignment."
    }
  }
}',
'{
  "type": "VerticalLayout",
  "elements": [
    {
      "type": "Group",
      "label": "Selectivity Identification",
      "elements": [
        { "type": "Control", "scope": "#/properties/warrantRef" },
        { "type": "Control", "scope": "#/properties/selectivityTimestamp" }
      ]
    },
    {
      "type": "Group",
      "label": "Risk Assessment Outcome",
      "elements": [
        { "type": "Control", "scope": "#/properties/riskLevel" },
        { "type": "Control", "scope": "#/properties/examinationRequired" },
        { "type": "Control", "scope": "#/properties/instructions" }
      ]
    }
  ]
}',
'1.0',
true);

INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES
('b1d0a101-0001-4000-8000-200000000012', 
'10ai: Logistics & Yard Entry', 
'Terminal Entry and Container Gating Verification', 
'{
  "type": "object",
  "title": "10ai: Yard Entry & Gating",
  "properties": {
    "warrantRef": {
      "type": "string",
      "title": "Warrant Number",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_warrant_no" }
    },
    "containerNumber": {
      "type": "string",
      "title": "Container Number",
      "x-globalcontext": { "writeTo": "global_container_no" },
      "default": "MSCU-882910-4"
    },
    "vehicleNumber": {
      "type": "string",
      "title": "Truck / Vehicle No",
      "default": "WP-LY-5521"
    },
    "assignedLane": {
      "type": "string",
      "title": "Customs Assigned Lane",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_assigned_lane" }
    },
    "declaredPackages": {
      "type": "integer",
      "title": "Declared Package Count",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_package_count" }
    },
    "declaredWeight": {
      "type": "number",
      "title": "Declared Weight (Kg)",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_total_weight" }
    },
    "gateInTimestamp": {
      "type": "string",
      "title": "Gate-In Time",
      "readOnly": true,
      "default": "2026-02-12 16:20:00"
    },
    "terminalLocation": {
      "type": "string",
      "title": "Yard Storage Location",
      "default": "BLOCK-C / SL-04"
    }
  }
}',
'{
  "type": "VerticalLayout",
  "elements": [
    {
      "type": "Group",
      "label": "Security & Reference",
      "elements": [
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/warrantRef" },
            { "type": "Control", "scope": "#/properties/assignedLane" }
          ]
        },
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/containerNumber" },
            { "type": "Control", "scope": "#/properties/vehicleNumber" }
          ]
        }
      ]
    },
    {
      "type": "Group",
      "label": "Physical Cargo Verification",
      "elements": [
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/declaredPackages" },
            { "type": "Control", "scope": "#/properties/declaredWeight" }
          ]
        },
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/gateInTimestamp" },
            { "type": "Control", "scope": "#/properties/terminalLocation" }
          ]
        }
      ]
    }
  ]
}',
'1.0',
true);

INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES
('b1d0a101-0001-4000-8000-200000000016', 
'10ci: Export Release', 
'Final Customs release (Boat Note) for vessel loading', 
'{
  "type": "object",
  "title": "10ci: Export Release (Boat Note)",
  "properties": {
    "cusdecRef": {
      "type": "string",
      "title": "Cusdec Reference",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_cusdec_no" }
    },
    "containerNo": {
      "type": "string",
      "title": "Container Identification",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_container_no" }
    },
    "boatNoteNumber": {
      "type": "string",
      "title": "Boat Note Number",
      "x-globalcontext": { "writeTo": "global_boat_note_no" },
      "default": "BN-2026-X883"
    },
    "vesselName": {
      "type": "string",
      "title": "Vessel Name / Voyage",
      "default": "MSC EMMA / V.2403"
    },
    "releaseTimestamp": {
      "type": "string",
      "title": "Customs Release Time",
      "readOnly": true,
      "default": "2026-02-12 18:00:00"
    },
    "customsOfficerID": {
      "type": "string",
      "title": "Authorizing Officer ID",
      "default": "SLC-OFF-093"
    },
    "status": {
      "type": "string",
      "title": "Release Status",
      "readOnly": true,
      "default": "EXPORT RELEASED"
    }
  }
}',
'{
  "type": "VerticalLayout",
  "elements": [
    {
      "type": "Group",
      "label": "Consignment Release Reference",
      "elements": [
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/cusdecRef" },
            { "type": "Control", "scope": "#/properties/containerNo" }
          ]
        }
      ]
    },
    {
      "type": "Group",
      "label": "Customs Boat Note Information",
      "elements": [
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/boatNoteNumber" },
            { "type": "Control", "scope": "#/properties/vesselName" }
          ]
        },
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/releaseTimestamp" },
            { "type": "Control", "scope": "#/properties/customsOfficerID" }
          ]
        },
        { "type": "Control", "scope": "#/properties/status" }
      ]
    }
  ]
}',
'1.0',
true);

INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES
('b1d0a101-0001-4000-8000-200000000018', 
'12: Bill of Lading', 
'Submission of final transport document from the Shipping Line', 
'{
  "type": "object",
  "title": "12: Bill of Lading (B/L) Submission",
  "properties": {
    "cusdecRef": {
      "type": "string",
      "title": "Cusdec Reference",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_cusdec_no" }
    },
    "containerNo": {
      "type": "string",
      "title": "Container Number",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_container_no" }
    },
    "billOfLadingNo": {
      "type": "string",
      "title": "Bill of Lading Number",
      "x-globalcontext": { "writeTo": "global_bl_no" },
      "default": "MSCU-CMB-1299"
    },
    "shippingLine": {
      "type": "string",
      "title": "Shipping Line",
      "default": "MSC - Mediterranean Shipping Company"
    },
    "vesselName": {
      "type": "string",
      "title": "Vessel Name / Voyage",
      "x-globalcontext": { "readFrom": "global_vessel_name" },
      "readOnly": true
    },
    "onBoardDate": {
      "type": "string",
      "format": "date",
      "title": "Shipped on Board Date",
      "x-globalcontext": { "writeTo": "global_onboard_date" },
      "default": "2026-02-13"
    },
    "portOfLoading": {
      "type": "string",
      "title": "Port of Loading",
      "default": "Colombo (LKCMB)"
    },
    "portOfDischarge": {
      "type": "string",
      "title": "Port of Discharge",
      "x-globalcontext": { "readFrom": "global_destination_country" },
      "readOnly": true
    },
    "ATTACHMENT_billOfLading": {
      "type": "string",
      "title": "ATTACHMENT: Final Bill of Lading (PDF)",
      "default": "bl_mscu_1299.pdf"
    }
  }
}',
'{
  "type": "VerticalLayout",
  "elements": [
    {
      "type": "Group",
      "label": "Shipment Linkage",
      "elements": [
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/cusdecRef" },
            { "type": "Control", "scope": "#/properties/containerNo" }
          ]
        },
        { "type": "Control", "scope": "#/properties/vesselName" }
      ]
    },
    {
      "type": "Group",
      "label": "Transport Document Details",
      "elements": [
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/billOfLadingNo" },
            { "type": "Control", "scope": "#/properties/shippingLine" }
          ]
        },
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/portOfLoading" },
            { "type": "Control", "scope": "#/properties/portOfDischarge" }
          ]
        },
        { "type": "Control", "scope": "#/properties/onBoardDate" },
        { "type": "Control", "scope": "#/properties/ATTACHMENT_billOfLading" }
      ]
    }
  ]
}',
'1.0',
true);

INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES
('b1d0a101-0001-4000-8000-200000000020', 
'8bi: Phyto Issuance', 
'Final Issuance of Phytosanitary Certificate based on B/L and Inspection', 
'{
  "type": "object",
  "title": "8bi: Phytosanitary Certificate Issuance",
  "properties": {
    "certificateNo": {
      "type": "string",
      "title": "Phyto Certificate Number",
      "x-globalcontext": { "writeTo": "global_phyto_cert_no" },
      "default": "PSC/LK/2026/08821"
    },
    "billOfLadingRef": {
      "type": "string",
      "title": "Associated B/L Number",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_bl_no" }
    },
    "shippedOnDate": {
      "type": "string",
      "title": "Vessel Departure Date",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_onboard_date" }
    },
    "botanicalName": {
      "type": "string",
      "title": "Botanical Name",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_botanical_name" }
    },
    "inspectionResult": {
      "type": "string",
      "title": "Inspection Findings",
      "readOnly": true,
      "default": "Consignment inspected and found free from quarantine pests."
    },
    "issuingOfficer": {
      "type": "string",
      "title": "Authorized NPQS Officer",
      "default": "Dr. S. Perera (NPQS)"
    },
    "ATTACHMENT_finalPhytoCert": {
      "type": "string",
      "title": "DOWNLOAD: Digital Phytosanitary Certificate (Signed)",
      "default": "phyto_cert_signed_final.pdf"
    }
  }
}',
'{
  "type": "VerticalLayout",
  "elements": [
    {
      "type": "Group",
      "label": "Certificate Identification",
      "elements": [
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/certificateNo" },
            { "type": "Control", "scope": "#/properties/issuingOfficer" }
          ]
        }
      ]
    },
    {
      "type": "Group",
      "label": "Validated Transport & Inspection Data",
      "elements": [
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/billOfLadingRef" },
            { "type": "Control", "scope": "#/properties/shippedOnDate" }
          ]
        },
        { "type": "Control", "scope": "#/properties/botanicalName" },
        { "type": "Control", "scope": "#/properties/inspectionResult" }
      ]
    },
    {
      "type": "Group",
      "label": "Final Document",
      "elements": [
        { "type": "Control", "scope": "#/properties/ATTACHMENT_finalPhytoCert" }
      ]
    }
  ]
}',
'1.0',
true);

INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES
('b1d0a101-0001-4000-8000-200000000011', 
'9-Iss: Health Issuance', 
'Final Issuance of Food Safety Health Certificate based on Lab Results and B/L', 
'{
  "type": "object",
  "title": "9-Iss: Health Certificate Issuance",
  "properties": {
    "healthCertNo": {
      "type": "string",
      "title": "Health Certificate Number",
      "x-globalcontext": { "writeTo": "global_health_cert_no" },
      "default": "H-CERT-2026-4412"
    },
    "cusdecRef": {
      "type": "string",
      "title": "Cusdec Reference",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_cusdec_no" }
    },
    "billOfLadingRef": {
      "type": "string",
      "title": "Bill of Lading Number",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_bl_no" }
    },
    "natureOfProduct": {
      "type": "string",
      "title": "Product Description",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_package_summary" }
    },
    "labTestResult": {
      "type": "string",
      "title": "Laboratory Analysis Result",
      "readOnly": true,
      "default": "PASSED - No Salmonella or contaminants detected."
    },
    "expiryDate": {
      "type": "string",
      "format": "date",
      "title": "Certificate Expiry Date",
      "default": "2026-08-12"
    },
    "issuingAuthority": {
      "type": "string",
      "title": "Issuing Authority",
      "default": "Food Control Administration Unit (FCAU)"
    },
    "ATTACHMENT_signedHealthCert": {
      "type": "string",
      "title": "DOWNLOAD: Digital Health Certificate (Signed)",
      "default": "health_cert_final_signed.pdf"
    }
  }
}',
'{
  "type": "VerticalLayout",
  "elements": [
    {
      "type": "Group",
      "label": "Health Certificate Identification",
      "elements": [
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/healthCertNo" },
            { "type": "Control", "scope": "#/properties/issuingAuthority" }
          ]
        },
        { "type": "Control", "scope": "#/properties/expiryDate" }
      ]
    },
    {
      "type": "Group",
      "label": "Consignment & Laboratory Verification",
      "elements": [
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/cusdecRef" },
            { "type": "Control", "scope": "#/properties/billOfLadingRef" }
          ]
        },
        { "type": "Control", "scope": "#/properties/natureOfProduct" },
        { "type": "Control", "scope": "#/properties/labTestResult" }
      ]
    },
    {
      "type": "Group",
      "label": "Final Approved Document",
      "elements": [
        { "type": "Control", "scope": "#/properties/ATTACHMENT_signedHealthCert" }
      ]
    }
  ]
}',
'1.0',
true);

INSERT INTO forms (id, name, description, schema, ui_schema, version, active) VALUES
('b1d0a101-0001-4000-8000-200000000019', 
'13: Country of Origin', 
'Final Issuance of Certificate of Origin for preferential/non-preferential trade', 
'{
  "type": "object",
  "title": "13: Certificate of Origin (COO)",
  "properties": {
    "cooNumber": {
      "type": "string",
      "title": "Certificate Number",
      "default": "COO-SL-2026-8812",
      "x-globalcontext": { "writeTo": "global_coo_no" }
    },
    "exporterDetails": {
      "type": "string",
      "title": "Exporter (Name, Address, Country)",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_exporter_name" }
    },
    "consigneeDetails": {
      "type": "string",
      "title": "Consignee (Name, Address, Country)",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_consignee_name" }
    },
    "transportDetails": {
      "type": "string",
      "title": "Transport Details (Vessel/V_No/BL_No)",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_bl_no" }
    },
    "itemDescription": {
      "type": "string",
      "title": "Description of Goods",
      "readOnly": true,
      "x-globalcontext": { "readFrom": "global_package_summary" }
    },
    "originCriteria": {
      "type": "string",
      "title": "Origin Criterion",
      "enum": ["Wholly Obtained (WO)", "Value Added (VA)", "Change in Tariff Heading (CTH)"],
      "default": "Wholly Obtained (WO)"
    },
    "issuingBody": {
      "type": "string",
      "title": "Issuing Authority",
      "default": "Department of Commerce, Sri Lanka"
    },
    "ATTACHMENT_finalCOO": {
      "type": "string",
      "title": "DOWNLOAD: Digital Certificate of Origin (Signed)",
      "default": "coo_final_signed.pdf"
    }
  }
}',
'{
  "type": "VerticalLayout",
  "elements": [
    {
      "type": "Group",
      "label": "Parties & Identification",
      "elements": [
        { "type": "Control", "scope": "#/properties/cooNumber" },
        { "type": "Control", "scope": "#/properties/exporterDetails" },
        { "type": "Control", "scope": "#/properties/consigneeDetails" }
      ]
    },
    {
      "type": "Group",
      "label": "Product & Origin Details",
      "elements": [
        { "type": "Control", "scope": "#/properties/itemDescription" },
        {
          "type": "HorizontalLayout",
          "elements": [
            { "type": "Control", "scope": "#/properties/transportDetails" },
            { "type": "Control", "scope": "#/properties/originCriteria" }
          ]
        },
        { "type": "Control", "scope": "#/properties/issuingBody" }
      ]
    },
    {
      "type": "Group",
      "label": "Certified Document",
      "elements": [
        { "type": "Control", "scope": "#/properties/ATTACHMENT_finalCOO" }
      ]
    }
  ]
}',
'1.0',
true);