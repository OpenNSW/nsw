import { Handle, Position } from '@xyflow/react'
import { Text, Tooltip } from '@radix-ui/themes'
import { PlusIcon, Cross2Icon, QuestionMarkIcon } from '@radix-ui/react-icons'
import type { WorkflowNodeV2 } from '../../services/types/workflow'
import { getStatusStyle } from './utils'

interface GatewayNodeProps {
  step: WorkflowNodeV2
  targetPosition?: Position
  sourcePosition?: Position
}

export function GatewayNode({ step, targetPosition = Position.Left, sourcePosition = Position.Right }: GatewayNodeProps) {
  const statusStyle = getStatusStyle(step.state)
  const isParallel = step.gateway_type?.includes('PARALLEL')
  const isExclusive = step.gateway_type?.includes('EXCLUSIVE')

  return (
    <div className="relative flex items-center justify-center">
      <Handle type="target" position={targetPosition} className="bg-slate-400! w-2! h-2!" />
      <Tooltip content={step.name}>
        <div
          className={`w-12 h-12 rotate-45 border-2 flex items-center justify-center bg-white ${statusStyle.borderColor} shadow-sm group hover:scale-110 transition-transform`}
        >
          <div className="-rotate-45 flex items-center justify-center">
            {isParallel ? (
              <PlusIcon className={`w-6 h-6 ${statusStyle.iconColor}`} />
            ) : isExclusive ? (
              <Cross2Icon className={`w-6 h-6 ${statusStyle.iconColor}`} />
            ) : (
              <QuestionMarkIcon className={`w-6 h-6 ${statusStyle.iconColor}`} />
            )}
          </div>
        </div>
      </Tooltip>
      <div className="absolute top-full mt-2 whitespace-nowrap">
        <Text size="1" weight="bold" color="gray">{step.name}</Text>
      </div>
      <Handle type="source" position={sourcePosition} className="bg-slate-400! w-2! h-2!" />
    </div>
  )
}
