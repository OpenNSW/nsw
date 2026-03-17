import { Handle, Position } from '@xyflow/react'
import { Text, Tooltip } from '@radix-ui/themes'
import { ShuffleIcon, QuestionMarkIcon } from '@radix-ui/react-icons'
import type { WorkflowNodeV2 } from '../../services/types/workflow'
import { getStatusStyle } from './utils'

interface GatewayNodeProps {
  step: WorkflowNodeV2
}

export function GatewayNode({ step }: GatewayNodeProps) {
  const statusStyle = getStatusStyle(step.state)
  const isSplit = step.gateway_type?.includes('SPLIT')

  return (
    <div className="relative flex items-center justify-center">
      <Handle type="target" position={Position.Left} className="bg-slate-400! w-2! h-2!" />
      <Tooltip content={step.name}>
        <div
          className={`w-12 h-12 rotate-45 border-2 flex items-center justify-center bg-white ${statusStyle.borderColor} shadow-sm group hover:scale-110 transition-transform`}
        >
          <div className="-rotate-45 flex items-center justify-center">
            {isSplit ? (
              <ShuffleIcon className={`w-6 h-6 ${statusStyle.iconColor}`} />
            ) : (
              <QuestionMarkIcon className={`w-6 h-6 ${statusStyle.iconColor}`} />
            )}
          </div>
        </div>
      </Tooltip>
      <div className="absolute top-full mt-2 whitespace-nowrap">
        <Text size="1" weight="bold" color="gray">{step.name}</Text>
      </div>
      <Handle type="source" position={Position.Right} className="bg-slate-400! w-2! h-2!" />
    </div>
  )
}
