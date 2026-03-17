import type { Node, NodeProps } from '@xyflow/react'
import type { WorkflowNodeV2 } from '../../services/types/workflow'
import type { WorkflowNode as LegacyWorkflowNode } from '../../services/types/consignment'
import { GatewayNode } from './GatewayNode'
import { EventNode } from './EventNode'
import { TaskNode } from './TaskNode'

export interface WorkflowNodeData extends Record<string, unknown> {
  step: WorkflowNodeV2 | LegacyWorkflowNode
}

export type WorkflowNodeType = Node<WorkflowNodeData, 'workflowStep'>

export function WorkflowNode({ data }: NodeProps<WorkflowNodeType>) {
  const { step } = data
  const isV2 = 'type' in step

  if (isV2) {
    const v2Step = step as WorkflowNodeV2
    if (v2Step.type === 'INTERNAL') {
      if (v2Step.internal_type === 'GATEWAY') {
        return <GatewayNode step={v2Step} />
      }
      if (v2Step.internal_type === 'EVENT') {
        return <EventNode step={v2Step} />
      }
    }
  }

  // TASK node or Legacy node
  return <TaskNode step={step} />
}
