import type { Workflow, WorkflowStep } from '../types/workflow'

// Sample workflow steps for tea import/export
const teaImportSteps: WorkflowStep[] = [
  {
    stepId: 'cusdec',
    type: 'TRADER_FORM',
    config: {
      formId: 'customs-declaration',
    },
    dependsOn: [],
  },
  {
    stepId: 'tea_board_permit',
    type: 'OGA_FORM',
    config: {
      agency: 'SLTB',
      service: 'import-permit',
    },
    dependsOn: ['cusdec'],
  },
  {
    stepId: 'quality_inspection',
    type: 'OGA_FORM',
    config: {
      agency: 'SLSI',
      service: 'quality-inspection',
    },
    dependsOn: ['cusdec'],
  },
  {
    stepId: 'customs_release',
    type: 'WAIT_FOR_EVENT',
    config: {
      event: 'CUSTOMS_RELEASED',
    },
    dependsOn: ['tea_board_permit', 'quality_inspection'],
  },
]

const teaExportSteps: WorkflowStep[] = [
  {
    stepId: 'cusdec_entry',
    type: 'TRADER_FORM',
    config: {
      formId: 'customs-declaration-export',
    },
    dependsOn: [],
  },
  {
    stepId: 'phytosanitary_cert',
    type: 'OGA_FORM',
    config: {
      agency: 'NPQS',
      service: 'plant-quarantine',
    },
    dependsOn: ['cusdec_entry'],
  },
  {
    stepId: 'tea_blend_sheet',
    type: 'OGA_FORM',
    config: {
      agency: 'SLTB',
      service: 'tea-blend-sheet',
    },
    dependsOn: ['cusdec_entry'],
  },
  {
    stepId: 'final_customs_clearance',
    type: 'WAIT_FOR_EVENT',
    config: {
      event: 'CUSTOMS_RELEASED',
    },
    dependsOn: ['phytosanitary_cert', 'tea_blend_sheet'],
  },
]

// Mock workflow interface for filtering (includes hsCode for mock filtering)
interface MockWorkflow extends Workflow {
  hsCode: string
}

// Workflows are associated with HS codes (the code string)
// When searching for a parent code, all child workflows are returned
export const mockWorkflows: MockWorkflow[] = [
  // 090210 - Green tea (not fermented) in immediate packings ≤3kg
  {
    id: 'wf-09021011-import',
    name: 'sl-import-tea-green-1.0',
    type: 'import',
    hsCode: '09021011',
    steps: teaImportSteps,
  },
  {
    id: 'wf-09021011-export',
    name: 'sl-export-tea-green-1.0',
    type: 'export',
    hsCode: '09021011',
    steps: teaExportSteps,
  },
  {
    id: 'wf-09021012-import',
    name: 'sl-import-tea-green-certified-1.0',
    type: 'import',
    hsCode: '09021012',
    steps: teaImportSteps,
  },
  {
    id: 'wf-09021012-export',
    name: 'sl-export-tea-green-certified-1.0',
    type: 'export',
    hsCode: '09021012',
    steps: teaExportSteps,
  },
  // 090230 - Black tea (fermented) in immediate packings ≤3kg
  {
    id: 'wf-09023011-import',
    name: 'sl-import-tea-black-1.0',
    type: 'import',
    hsCode: '09023011',
    steps: teaImportSteps,
  },
  {
    id: 'wf-09023011-export',
    name: 'sl-export-tea-black-1.0',
    type: 'export',
    hsCode: '09023011',
    steps: teaExportSteps,
  },
  {
    id: 'wf-09023021-import',
    name: 'sl-import-tea-black-packaged-1.0',
    type: 'import',
    hsCode: '09023021',
    steps: teaImportSteps,
  },
  {
    id: 'wf-09023021-export',
    name: 'sl-export-tea-black-packaged-1.0',
    type: 'export',
    hsCode: '09023021',
    steps: teaExportSteps,
  },
  // 090240 - Other black tea (fermented)
  {
    id: 'wf-09024011-import',
    name: 'sl-import-tea-black-bulk-1.0',
    type: 'import',
    hsCode: '09024011',
    steps: teaImportSteps,
  },
  {
    id: 'wf-09024011-export',
    name: 'sl-export-tea-black-bulk-1.0',
    type: 'export',
    hsCode: '09024011',
    steps: teaExportSteps,
  },
]