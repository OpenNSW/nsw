import { Handle, Position } from '@xyflow/react'
import { Text, Tooltip } from '@radix-ui/themes'
import type { WorkflowNodeV2 } from '../../services/types/workflow'

interface EventNodeProps {
  step: WorkflowNodeV2
}

export function EventNode({ step }: EventNodeProps) {
  const isStart = step.event_type === 'START'

  return (
    <div className="flex flex-col items-center">
      <Handle type="target" position={Position.Left} className="bg-slate-400! w-2! h-2!" />
      <Tooltip content={step.name}>
        <div
          className={`w-10 h-10 rounded-full border-2 flex items-center justify-center ${isStart ? 'bg-emerald-50 border-emerald-400' : 'bg-red-50 border-red-400'
            } shadow-sm`}
        >
          <div className={`w-3 h-3 rounded-full ${isStart ? 'bg-emerald-500' : 'bg-red-500'}`} />
        </div>
      </Tooltip>
      <Text size="1" weight="bold" color="gray" className="mt-1">{step.name}</Text>
      <Handle type="source" position={Position.Right} className="bg-slate-400! w-2! h-2!" />
    </div>
  )
}
