import type React from 'react'
import {
  FileTextIcon,
  ReaderIcon,
  ClockIcon,
  CheckCircledIcon,
  LockClosedIcon,
} from '@radix-ui/react-icons'
import type { WorkflowNodeState, WorkflowNode as LegacyWorkflowNode } from '../../services/types/consignment'
import type { WorkflowNodeV2 } from '../../services/types/workflow'

export const legacyTypeIcons: Record<string, React.ReactNode> = {
  SIMPLE_FORM: <FileTextIcon className="w-4 h-4" />,
  WAIT_FOR_EVENT: <ClockIcon className="w-4 h-4" />,
  DOCUMENT_UPLOAD: <ReaderIcon className="w-4 h-4" />,
}

export interface StatusStyle {
  bgColor: string
  borderColor: string
  textColor: string
  iconColor: string
  statusIcon?: React.ReactNode
}

export const statusConfig: Record<WorkflowNodeState, StatusStyle> = {
  COMPLETED: {
    bgColor: 'bg-emerald-50',
    borderColor: 'border-emerald-400',
    textColor: 'text-emerald-700',
    iconColor: 'text-emerald-600',
    statusIcon: <CheckCircledIcon className="w-4 h-4 text-emerald-600" />,
  },
  READY: {
    bgColor: 'bg-blue-50',
    borderColor: 'border-blue-400',
    textColor: 'text-blue-700',
    iconColor: 'text-blue-600',
  },
  IN_PROGRESS: {
    bgColor: 'bg-orange-50',
    borderColor: 'border-orange-400',
    textColor: 'text-orange-700',
    iconColor: 'text-orange-600',
  },
  LOCKED: {
    bgColor: 'bg-slate-100',
    borderColor: 'border-slate-300',
    textColor: 'text-slate-500',
    iconColor: 'text-slate-400',
    statusIcon: <LockClosedIcon className="w-3 h-3 text-slate-400" />,
  },
  REJECTED: {
    bgColor: 'bg-red-50',
    borderColor: 'border-red-400',
    textColor: 'text-red-700',
    iconColor: 'text-red-600',
  },
}

export const getStatusStyle = (state?: WorkflowNodeState): StatusStyle => {
  return statusConfig[state || 'LOCKED'] || {
    bgColor: 'bg-gray-50',
    borderColor: 'border-gray-300',
    textColor: 'text-gray-500',
    iconColor: 'text-gray-400'
  }
}

export const getStepLabel = (step: WorkflowNodeV2 | LegacyWorkflowNode) => {
  if ('type' in step) return (step as WorkflowNodeV2).name || step.id
  
  const legacyStep = step as LegacyWorkflowNode
  if (legacyStep.workflowNodeTemplate.name) {
    return legacyStep.workflowNodeTemplate.name
  }
  const parts = legacyStep.id.split('-')
  const lastPart = parts[parts.length - 1]
  return `Step ${lastPart}`
}

export const getTooltipContent = (step: WorkflowNodeV2 | LegacyWorkflowNode) => {
  if ('type' in step) return getStepLabel(step)
  
  const legacyStep = step as LegacyWorkflowNode
  const label = getStepLabel(step)
  const description = legacyStep.workflowNodeTemplate.description

  if (description && description.trim()) {
    return `${label} - ${description}`
  }
  return label
}
