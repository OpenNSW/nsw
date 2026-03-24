import React, {useState} from 'react'
import {Handle, Position} from '@xyflow/react'
import {Text, Tooltip} from '@radix-ui/themes'
import {useParams, useNavigate} from 'react-router-dom'
import {
  PlayIcon,
  UpdateIcon,
  FileTextIcon,
  ReaderIcon,
} from '@radix-ui/react-icons'
import type {WorkflowNodeV2} from '../../services/types/workflow'
import type {WorkflowNode as LegacyWorkflowNode} from '../../services/types/consignment'
import {getStatusStyle, getStepLabel, getTooltipContent, legacyTypeIcons} from './utils'

interface TaskNodeProps {
  step: WorkflowNodeV2 | LegacyWorkflowNode
  targetPosition?: Position
  sourcePosition?: Position
}

export function TaskNode({step, targetPosition = Position.Left, sourcePosition = Position.Right}: TaskNodeProps) {
  const {consignmentId} = useParams<{ consignmentId: string }>()
  const navigate = useNavigate()
  const [isLoading, setIsLoading] = useState(false)

  const statusStyle = getStatusStyle(step.state)
  const isV2Step = 'type' in step
  const isExecutable = step.state === 'READY'
  const isViewable = step.state !== 'LOCKED' && !isExecutable && (!isV2Step || (step as WorkflowNodeV2).type === 'TASK')

  const getViewButtonColors = () => {
    switch (step.state) {
      case 'COMPLETED':
        return 'bg-emerald-500 hover:bg-emerald-600 active:bg-emerald-700'
      case 'IN_PROGRESS':
        return 'bg-orange-500 hover:bg-orange-600 active:bg-orange-700'
      case 'REJECTED':
        return 'bg-red-500 hover:bg-red-600 active:bg-red-700'
      default:
        return 'bg-slate-500 hover:bg-slate-600 active:bg-slate-700'
    }
  }

  const handleOpen = (e: React.MouseEvent) => {
    e.stopPropagation()
    if (!consignmentId) {
      console.error('No consignment ID found in URL')
      return
    }

    setIsLoading(true)
    navigate(`/consignments/${consignmentId}/tasks/${step.id}`)
  }

  const icon = isV2Step
    ? <FileTextIcon className="w-3.5 h-3.5"/>
    : legacyTypeIcons[(step as LegacyWorkflowNode).workflowNodeTemplate.type] || <FileTextIcon className="w-3.5 h-3.5"/>

  return (
    <div
      className={`px-3 py-2 rounded-lg border-2 hover:cursor-default shadow-sm w-72 min-h-[80px] flex flex-col justify-center ${statusStyle.bgColor
      } ${statusStyle.borderColor} ${step.state === 'READY' ? 'ring-2 ring-blue-300 ring-offset-2' : ''
      }`}
    >
      <Handle
        type="target"
        position={targetPosition}
        className="bg-slate-400! w-3! h-3!"
      />
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-start gap-2 flex-1 min-w-0">
          <div className={`mt-0.5 shrink-0 ${statusStyle.iconColor}`}>
            {icon}
          </div>
          <div className="min-w-0">
            <Tooltip content={getTooltipContent(step)}>
              <Text
                size="1"
                weight="bold"
                className={`${statusStyle.textColor} block cursor-pointer whitespace-normal break-words leading-tight`}
              >
                {getStepLabel(step)}
              </Text>
            </Tooltip>
            <div>
              <Text size="1" className={`${statusStyle.textColor} font-mono mt-0.5 text-xs`}>
                {step.state}
              </Text>
            </div>
            <div>
              {step.state === "IN_PROGRESS" &&
                  <div className={`${statusStyle.textColor} mt-0.5 text-[0.5rem] italic`}>
                    {step.extendedState}
                  </div>
              }
            </div>
          </div>
        </div>

        {isExecutable && (
          <button
            onClick={handleOpen}
            disabled={isLoading}
            className="flex items-center justify-center w-8 h-8 rounded-full bg-blue-500 hover:bg-blue-600 active:bg-blue-700 text-white shadow-md hover:cursor-pointer hover:shadow-lg transition-all duration-150 shrink-0 disabled:bg-slate-400 disabled:cursor-not-allowed"
            title="Execute task"
          >
            {isLoading ? (
              <UpdateIcon className="w-4 h-4 animate-spin"/>
            ) : (
              <PlayIcon className="w-4 h-4 ml-0.5"/>
            )}
          </button>
        )}

        {isViewable && (
          <button
            onClick={handleOpen}
            disabled={isLoading}
            className={`flex items-center justify-center w-8 h-8 rounded-full ${getViewButtonColors()} text-white shadow-md hover:cursor-pointer hover:shadow-lg transition-all duration-150 shrink-0 disabled:bg-slate-400 disabled:cursor-not-allowed`}
            title="View task"
          >
            {isLoading ? (
              <UpdateIcon className="w-4 h-4 animate-spin"/>
            ) : (
              <ReaderIcon className="w-4 h-4"/>
            )}
          </button>
        )}
      </div>

      <Handle
        type="source"
        position={sourcePosition}
        className="bg-slate-400! w-3! h-3!"
      />
    </div>
  )
}
