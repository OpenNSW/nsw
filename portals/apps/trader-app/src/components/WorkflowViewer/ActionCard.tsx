import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Text,
  Badge,
  Button,
  Card,
  Flex,
  Box,
} from '@radix-ui/themes'
import {
  CheckCircledIcon,
  PlayIcon,
  UpdateIcon,
  FileTextIcon,
  ClockIcon,
  ReaderIcon,
  InfoCircledIcon,
} from '@radix-ui/react-icons'
import type { WorkflowNode, WorkflowNodeState } from '../../services/types/consignment'

const nodeTypeIcons: Record<string, React.ReactNode> = {
  SIMPLE_FORM: <FileTextIcon className="w-4 h-4" />,
  WAIT_FOR_EVENT: <ClockIcon className="w-4 h-4" />,
  PAYMENT: <ReaderIcon className="w-4 h-4" />,
  DOCUMENT_UPLOAD: <ReaderIcon className="w-4 h-4" />,
}

const statusConfig: Record<
  WorkflowNodeState,
  {
    color: 'green' | 'blue' | 'orange' | 'gray' | 'red'
    label: string
    icon: React.ReactNode
  }
> = {
  COMPLETED: {
    color: 'green',
    label: 'Completed',
    icon: <CheckCircledIcon className="w-4 h-4" />,
  },
  READY: {
    color: 'blue',
    label: 'Ready',
    icon: <PlayIcon className="w-4 h-4" />,
  },
  IN_PROGRESS: {
    color: 'orange',
    label: 'In Progress',
    icon: <UpdateIcon className="w-4 h-4" />,
  },
  LOCKED: {
    color: 'gray',
    label: 'Locked',
    icon: <ClockIcon className="w-3 h-3" />, // Using Clock for locked as well or keep LockClosed
  },
  REJECTED: {
    color: 'red',
    label: 'Rejected',
    icon: <CheckCircledIcon className="w-4 h-4" />,
  },
}

// Re-adding LockClosedIcon for the actual Locked state
import { LockClosedIcon } from '@radix-ui/react-icons'
statusConfig.LOCKED.icon = <LockClosedIcon className="w-3 h-3" />

export interface ActionCardProps {
  step: WorkflowNode
  consignmentId: string
}

export const ActionCard = ({ step, consignmentId }: ActionCardProps) => {
  const navigate = useNavigate()
  const [isLoading, setIsLoading] = useState(false)
  const config = statusConfig[step.state] || { color: 'gray', label: step.state, icon: null }

  const handleOpen = async () => {
    setIsLoading(true)
    try {
      navigate(`/consignments/${consignmentId}/tasks/${step.id}`)
    } catch (error) {
      console.error('Failed to navigate to task:', error)
    } finally {
      setIsLoading(false)
    }
  }

  const label = step.workflowNodeTemplate.name || `Step ${step.id.split('-').pop()}`
  const isExecutable = step.state === 'READY'
  const isViewable = step.state !== 'LOCKED' && !isExecutable

  return (
    <Card variant="classic" className="mb-3 hover:shadow-lg transition-all duration-200 bg-white border border-gray-100 shadow-sm">
      <Flex direction="column" gap="3">
        <Flex align="start" justify="between" gap="3">
          <Flex align="center" gap="3" className="flex-1 min-w-0">
            <Box className={`p-2.5 rounded-lg bg-${config.color}-50 text-${config.color}-600 border border-${config.color}-100`}>
              {nodeTypeIcons[step.workflowNodeTemplate.type] || <FileTextIcon className="w-5 h-5" />}
            </Box>
            <Box className="flex-1 min-w-0">
              <Text size="3" weight="bold" className="block truncate text-gray-900">
                {label}
              </Text>
              <Flex align="center" gap="2" mt="1">
                <Badge color={config.color} variant="soft" size="1">
                  <Flex align="center" gap="1">
                    {config.icon}
                    {config.label}
                  </Flex>
                </Badge>
                {step.workflowNodeTemplate.type && (
                   <Text size="1" color="gray" className="uppercase tracking-wider font-medium opacity-70">
                     • {step.workflowNodeTemplate.type.replace(/_/g, ' ')}
                   </Text>
                )}
              </Flex>
            </Box>
          </Flex>

          <Box>
            {isExecutable && (
              <Button
                size="2"
                onClick={handleOpen}
                disabled={isLoading}
                loading={isLoading}
                className="cursor-pointer"
              >
                <PlayIcon />
                Start Task
              </Button>
            )}
            {isViewable && (
              <Button
                variant="soft"
                color="gray"
                size="2"
                onClick={handleOpen}
                disabled={isLoading}
                loading={isLoading}
                className="cursor-pointer"
              >
                <ReaderIcon />
                View Details
              </Button>
            )}
          </Box>
        </Flex>

        {step.workflowNodeTemplate.description && (
          <Box className="bg-gray-50/50 p-2 rounded border border-gray-100/50">
             <Text size="2" color="gray" className="leading-relaxed">
               {step.workflowNodeTemplate.description}
             </Text>
          </Box>
        )}

        {step.extendedState && (
           <Flex align="center" gap="1" className="text-orange-600">
              <InfoCircledIcon className="w-3.5 h-3.5" />
              <Text size="1" weight="medium" className="italic">
                {step.extendedState}
              </Text>
           </Flex>
        )}
      </Flex>
    </Card>
  )
}