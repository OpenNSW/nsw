import { useState } from 'react'
import { WorkflowViewer } from '../components/WorkflowViewer'
import { TextArea, Button, Box, Flex, Text, Heading, Badge } from '@radix-ui/themes'
import { CopyIcon, CheckIcon, ExclamationTriangleIcon } from '@radix-ui/react-icons'

const INITIAL_WORKFLOW = {
  "workflow_id": "customs-export-v1",
  "name": "Customs Export Declaration & Release",
  "version": 1,
  "edges": [
    {
      "id": "e_customs_start",
      "source_id": "customs_0_start",
      "target_id": "customs_1_cusdec_submit"
    },
    {
      "id": "e_customs_submit_to_pay",
      "source_id": "customs_1_cusdec_submit",
      "target_id": "customs_2_duty_payment"
    },
    {
      "id": "e_customs_pay_to_warrant",
      "source_id": "customs_2_duty_payment",
      "target_id": "customs_3_warranting_gw"
    },
    {
      "id": "e_customs_warrant_lcl",
      "source_id": "customs_3_warranting_gw",
      "target_id": "customs_4_lcl_cdn_create",
      "condition": "consignment_type == 'LCL'"
    },
    {
      "id": "e_customs_warrant_fcl",
      "source_id": "customs_3_warranting_gw",
      "target_id": "customs_4_fcl_cdn_create",
      "condition": "consignment_type == 'FCL'"
    },
    {
      "id": "e_customs_lcl_ack",
      "source_id": "customs_4_lcl_cdn_create",
      "target_id": "customs_5_cdn_ack"
    },
    {
      "id": "e_customs_fcl_ack",
      "source_id": "customs_4_fcl_cdn_create",
      "target_id": "customs_5_cdn_ack"
    },
    {
      "id": "e_customs_ack_bn_create",
      "source_id": "customs_5_cdn_ack",
      "target_id": "customs_6_boatnote_create"
    },
    {
      "id": "e_customs_bn_create_to_appr",
      "source_id": "customs_6_boatnote_create",
      "target_id": "customs_6_boatnote_approve"
    },
    {
      "id": "e_customs_bn_done",
      "source_id": "customs_6_boatnote_approve",
      "target_id": "customs_7_export_released"
    }
  ],
  "nodes": [
    {
      "id": "customs_0_start",
      "type": "INTERNAL",
      "name": "START",
      "internal_type": "EVENT",
      "event_type": "START",
      "x": -53,
      "y": 125
    },
    {
      "id": "customs_1_cusdec_submit",
      "type": "TASK",
      "name": "[Customs 1.1] Submit CusDec",
      "task_id": "SUBMIT_CUSDEC",
      "output_mapping": {
        "consignment_type": "consignment_type"
      },
      "x": 237,
      "y": 136
    },
    {
      "id": "customs_2_duty_payment",
      "type": "TASK",
      "name": "[Customs 2.1] Pay Cess / Duties",
      "task_id": "PAY_DUTIES",
      "x": 613,
      "y": 142
    },
    {
      "id": "customs_3_warranting_gw",
      "type": "INTERNAL",
      "name": "[Customs 3.1] Warranting (Logic)",
      "internal_type": "GATEWAY",
      "gateway_type": "EXCLUSIVE_SPLIT",
      "x": 980,
      "y": 364
    },
    {
      "id": "customs_4_lcl_cdn_create",
      "type": "TASK",
      "name": "[Customs 4.1] LCL: Create CDN (CFS)",
      "task_id": "CREATE_LCL_CDN",
      "x": 1364,
      "y": 592
    },
    {
      "id": "customs_4_fcl_cdn_create",
      "type": "TASK",
      "name": "[Customs 4.2] FCL: Create CDN",
      "task_id": "CREATE_FCL_CDN",
      "x": 1306,
      "y": 75
    },
    {
      "id": "customs_5_cdn_ack",
      "type": "TASK",
      "name": "[Customs 5.1] All CDN(s) Acked",
      "task_id": "ACK_CDNS",
      "x": 1736,
      "y": 121
    },
    {
      "id": "customs_6_boatnote_create",
      "type": "TASK",
      "name": "[Customs 6.1] Create Boat Note",
      "task_id": "CREATE_BOAT_NOTE",
      "x": 2101,
      "y": 119
    },
    {
      "id": "customs_6_boatnote_approve",
      "type": "TASK",
      "name": "[Customs 6.2] Approve Boat Note",
      "task_id": "APPROVE_BOAT_NOTE",
      "x": 2475,
      "y": 126
    },
    {
      "id": "customs_7_export_released",
      "type": "INTERNAL",
      "name": "[Customs 7.1] Export Released",
      "internal_type": "EVENT",
      "event_type": "END",
      "x": 2863,
      "y": 97
    }
  ]
}

export function TestWorkflowScreen() {
  const [jsonValue, setJsonValue] = useState(JSON.stringify(INITIAL_WORKFLOW, null, 2))
  const [workflow, setWorkflow] = useState<any>(INITIAL_WORKFLOW)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const handleJsonChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const value = e.target.value
    setJsonValue(value)
    try {
      const parsed = JSON.parse(value)
      setWorkflow(parsed)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Invalid JSON')
    }
  }

  const handleCopy = () => {
    navigator.clipboard.writeText(jsonValue)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="w-screen h-screen bg-slate-50 flex flex-col p-6 overflow-hidden">
      <Flex direction="column" gap="4" className="h-full">
        <Flex justify="between" align="end">
          <Box>
            <Heading size="7" weight="bold" className="text-slate-900">Workflow Playground</Heading>
            <Text color="gray" size="2">Paste your BPMN configuration JSON below to visualize it.</Text>
          </Box>
          <Flex align="center" gap="3">
            {workflow?.workflow_id && (
              <Badge color="blue" variant="soft">
                {workflow.workflow_id} v{workflow.version || 1}
              </Badge>
            )}
            <Button variant="ghost" color="gray" onClick={() => window.location.href = '/login'}>
              Back to Login
            </Button>
          </Flex>
        </Flex>

        <Flex gap="4" className="flex-1 overflow-hidden">
          {/* JSON Input Panel */}
          <Flex direction="column" gap="2" className="w-[400px] h-full">
            <Flex justify="between" align="center">
              <Text size="2" weight="bold">Configuration JSON</Text>
              <Button size="1" variant="soft" onClick={handleCopy}>
                {copied ? <CheckIcon /> : <CopyIcon />}
                {copied ? 'Copied' : 'Copy'}
              </Button>
            </Flex>
            <Box className="flex-1 relative">
              <TextArea
                value={jsonValue}
                onChange={handleJsonChange}
                placeholder="Paste workflow JSON here..."
                className="w-full h-full font-mono text-xs resize-none border-slate-200"
                style={{ height: '100%' }}
              />
              {error && (
                <Flex 
                  align="center" 
                  gap="2" 
                  className="absolute bottom-2 left-2 right-2 p-2 bg-red-50 border border-red-200 rounded text-red-600 text-xs"
                >
                  <ExclamationTriangleIcon />
                  <Text>{error}</Text>
                </Flex>
              )}
            </Box>
          </Flex>

          {/* Visualization Panel */}
          <Box className="flex-1 bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden relative">
            {workflow ? (
              <WorkflowViewer workflow={workflow} className="w-full h-full" />
            ) : (
              <Flex align="center" justify="center" className="w-full h-full text-slate-400">
                <Text>Please enter a valid workflow configuration</Text>
              </Flex>
            )}
          </Box>
        </Flex>
      </Flex>
    </div>
  )
}
