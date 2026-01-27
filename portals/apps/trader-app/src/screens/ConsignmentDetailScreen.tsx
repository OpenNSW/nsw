import { useEffect, useState } from 'react'
import { useParams, useNavigate, useLocation } from 'react-router-dom'
import { Button, Badge, Spinner, Text, Flex, Progress, Box, Card } from '@radix-ui/themes'
import { ArrowLeftIcon, ChevronDownIcon, CheckCircledIcon } from '@radix-ui/react-icons'
import * as Accordion from '@radix-ui/react-accordion'
import { WorkflowViewer } from '../components/WorkflowViewer'
import type { Consignment } from "../services/types/consignment.ts"
import { getConsignment } from "../services/consignment.ts"
import { getStateColor, formatState } from '../utils/consignmentUtils'

export function ConsignmentDetailScreen() {
  const { consignmentId } = useParams<{ consignmentId: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const [consignment, setConsignment] = useState<Consignment | null>(null)
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [openAccordionItems, setOpenAccordionItems] = useState<string[]>([])

  const fetchConsignment = async () => {
    if (!consignmentId) {
      setError('Consignment ID is required')
      setLoading(false)
      return
    }

    setLoading(true)
    setError(null)
    try {
      const result = await getConsignment(consignmentId)
      if (result) {
        setConsignment(result)
      } else {
        setError('Consignment not found')
      }
    } catch (err) {
      console.error('Failed to fetch consignment:', err)
      setError('Failed to load consignment')
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }

  const handleRefresh = () => {
    setRefreshing(true)
    fetchConsignment()
  }

  useEffect(() => {
    // Check if we just submitted a form
    const state = location.state as { justSubmitted?: boolean } | null
    if (state?.justSubmitted) {
      // Clear the navigation state to prevent re-triggering on refresh
      navigate(location.pathname, { replace: true, state: {} })

      // Show loading state and fetch immediately
      setLoading(true)
      fetchConsignment()
    } else {
      // Normal fetch without delay
      fetchConsignment()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [consignmentId])

  // Set default open accordion item when consignment is loaded
  useEffect(() => {
    if (consignment && consignment.items.length === 1 && openAccordionItems.length === 0) {
      setOpenAccordionItems(['item-0'])
    }
  }, [consignment, openAccordionItems])

  if (loading) {
    const isProcessing = !consignment // If we don't have consignment data yet, we're in initial load
    return (
      <div className="p-6">
        <div className="flex items-center justify-center py-12">
          <Spinner size="3" />
          <Text size="3" color="gray" className="ml-3">
            {isProcessing ? 'Processing your submission...' : 'Loading consignment...'}
          </Text>
        </div>
      </div>
    )
  }

  if (error || !consignment) {
    return (
      <div className="p-6">
        <div className="bg-white rounded-lg shadow p-6 text-center">
          <Text size="4" color="red" weight="medium">
            {error || 'Consignment not found'}
          </Text>
          <div className="mt-4">
            <Button variant="soft" onClick={() => navigate('/consignments')}>
              <ArrowLeftIcon />
              Back to Consignments
            </Button>
          </div>
        </div>
      </div>
    )
  }

  const items = consignment.items
  const allSteps = items.flatMap(item => item.steps)
  const completedSteps = allSteps.filter(s => s.status === 'COMPLETED').length
  const totalSteps = allSteps.length

  return (
    <div className="p-6">
      <div className="mb-6">
        <Button variant="ghost" color="gray" onClick={() => navigate('/consignments')}>
          <ArrowLeftIcon />
          Back
        </Button>
      </div>

      <div className="bg-white rounded-lg shadow">
        <div className="p-6 border-b border-gray-200">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-semibold text-gray-900">
                Consignment
              </h1>
              <p className="mt-1 text-sm text-gray-500 font-mono">
                {consignment.id}
              </p>
              <p className="mt-1 text-sm text-gray-500">
                Created on {(() => {
                  const date = new Date(consignment.createdAt)
                  return !isNaN(date.getTime())
                    ? date.toLocaleDateString('en-US', {
                      year: 'numeric',
                      month: 'long',
                      day: 'numeric',
                      hour: '2-digit',
                      minute: '2-digit',
                    })
                    : '-'
                })()}
              </p>
            </div>
            <div className="flex flex-col items-end gap-2">
              <Badge size="2" color={getStateColor(consignment.state)}>
                {formatState(consignment.state)}
              </Badge>
              <Badge size="1" color={consignment.tradeFlow === 'IMPORT' ? 'blue' : 'green'} variant="soft">
                {consignment.tradeFlow}
              </Badge>
            </div>
          </div>
        </div>

        <div className="p-6 bg-gray-50">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
            <Card className="p-4">
              <Text size="2" color="gray" weight="medium" className="block mb-1">
                Total Steps
              </Text>
              <Text size="6" weight="bold" className="block">
                {totalSteps}
              </Text>
            </Card>
            <Card className="p-4">
              <Text size="2" color="gray" weight="medium" className="block mb-1">
                Completed
              </Text>
              <Flex align="center" gap="2">
                <Text size="6" weight="bold" className="block text-green-600">
                  {completedSteps}
                </Text>
                {completedSteps === totalSteps && totalSteps > 0 && (
                  <CheckCircledIcon className="text-green-600" width="24" height="24" />
                )}
              </Flex>
            </Card>
            <Card className="p-4">
              <Text size="2" color="gray" weight="medium" className="block mb-1">
                Items
              </Text>
              <Text size="6" weight="bold" className="block">
                {items.length}
              </Text>
            </Card>
          </div>

          <div className="mb-6">
            <Flex justify="between" align="center" mb="2">
              <Text size="2" weight="medium" color="gray">
                Overall Progress
              </Text>
              <Text size="2" weight="medium">
                {totalSteps > 0 ? Math.round((completedSteps / totalSteps) * 100) : 0}%
              </Text>
            </Flex>
            <Progress 
              value={totalSteps > 0 ? (completedSteps / totalSteps) * 100 : 0} 
              size="3"
              className="h-3"
            />
          </div>

          <div>
            <h3 className="text-sm font-medium text-gray-700 mb-3">
              {items.length === 1 ? 'HS Code' : 'HS Codes'}
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              {items.map((item, index) => {
                const itemSteps = item.steps || []
                const itemCompleted = itemSteps.filter(s => s.status === 'COMPLETED').length
                const itemTotal = itemSteps.length
                const itemProgress = itemTotal > 0 ? (itemCompleted / itemTotal) * 100 : 0

                return (
                  <Card key={index} className="p-4 hover:shadow-md transition-shadow">
                    <Flex justify="between" align="start" mb="2">
                      <Box>
                        {item.hsCode ? (
                          <>
                            <Text size="3" weight="bold" className="block mb-1">
                              {item.hsCode}
                            </Text>
                            <Text size="1" color="gray" className="block">
                              {item.hsCodeDescription}
                            </Text>
                          </>
                        ) : (
                          <Text size="3" weight="bold">{item.hsCodeID}</Text>
                        )}
                      </Box>
                      <Badge 
                        size="1" 
                        color={itemProgress === 100 ? 'green' : itemProgress > 0 ? 'blue' : 'gray'}
                        variant="soft"
                      >
                        {itemCompleted}/{itemTotal}
                      </Badge>
                    </Flex>
                    <Progress value={itemProgress} size="1" className="mt-2" />
                  </Card>
                )
              })}
            </div>
          </div>
        </div>

        {items.map((item, index) => {
          const itemSteps = item.steps || []
          if (itemSteps.length === 0) return null

          const itemCompletedSteps = itemSteps.filter(s => s.status === 'COMPLETED').length
          const itemTotalSteps = itemSteps.length

          return (
            <Accordion.Root
              key={index}
              type="multiple"
              value={openAccordionItems}
              onValueChange={setOpenAccordionItems}
              className="border-t border-gray-200"
            >
              <Accordion.Item value={`item-${index}`} className="group/item">
                <Accordion.Header>
                  <Accordion.Trigger className="w-full px-6 py-5 flex items-center justify-between hover:bg-blue-50 transition-all duration-200 group border-l-4 border-l-transparent hover:border-l-blue-500">
                    <div className="flex-1 text-left">
                      <Flex align="center" gap="3" mb="2">
                        {items.length > 1 && (
                          <Badge size="1" variant="soft" color="gray">
                            Item {index + 1}
                          </Badge>
                        )}
                        <h3 className="text-base font-semibold text-gray-900">
                          {item.hsCode || item.hsCodeID}
                        </h3>
                        <Badge
                          size="2"
                          color={itemCompletedSteps === itemTotalSteps ? 'green' : itemCompletedSteps > 0 ? 'blue' : 'gray'}
                          variant="soft"
                        >
                          {itemCompletedSteps}/{itemTotalSteps} steps
                        </Badge>
                      </Flex>
                      {item.hsCodeDescription && (
                        <Text size="2" color="gray" className="block">{item.hsCodeDescription}</Text>
                      )}
                      <div className="mt-2 w-48">
                        <Progress 
                          value={itemTotalSteps > 0 ? (itemCompletedSteps / itemTotalSteps) * 100 : 0} 
                          size="1"
                        />
                      </div>
                    </div>
                    <ChevronDownIcon
                      className="text-gray-400 group-hover:text-blue-600 transition-all duration-200 group-data-[state=open]:rotate-180 ml-4"
                      width="24"
                      height="24"
                    />
                  </Accordion.Trigger>
                </Accordion.Header>
                <Accordion.Content className="overflow-hidden data-[state=open]:animate-accordion-down data-[state=closed]:animate-accordion-up bg-gray-50">
                  <div className="px-6 py-6">
                    <div className="bg-white rounded-lg p-4 shadow-sm">
                      <Flex justify="between" align="center" mb="4">
                        <h4 className="text-sm font-semibold text-gray-700">Workflow Process</h4>
                        <Text size="1" color="gray">
                          {itemCompletedSteps} of {itemTotalSteps} completed
                        </Text>
                      </Flex>
                      <WorkflowViewer steps={itemSteps} onRefresh={handleRefresh} refreshing={refreshing} />
                    </div>
                  </div>
                </Accordion.Content>
              </Accordion.Item>
            </Accordion.Root>
          )
        })}

        <div className="p-6 border-t-2 border-gray-200">
          <Card className="p-5" style={{ backgroundColor: allSteps.every(s => s.status === 'COMPLETED') && allSteps.length > 0 ? '#f0fdf4' : '#f8fafc' }}>
            <Flex align="start" gap="3">
              <div className="mt-1">
                {allSteps.every(s => s.status === 'COMPLETED') && allSteps.length > 0 ? (
                  <CheckCircledIcon className="text-green-600" width="20" height="20" />
                ) : (
                  <div className="w-5 h-5 rounded-full bg-blue-100 flex items-center justify-center">
                    <span className="text-blue-600 text-xs font-bold">!</span>
                  </div>
                )}
              </div>
              <Box>
                <Text size="3" weight="bold" className="block mb-2" style={{ color: allSteps.every(s => s.status === 'COMPLETED') && allSteps.length > 0 ? '#16a34a' : '#1e293b' }}>
                  {allSteps.every(s => s.status === 'COMPLETED') && allSteps.length > 0 ? 'Consignment Complete!' : 'Next Steps'}
                </Text>
                {allSteps.some(s => s.status === 'READY') ? (
                  <Text size="2" color="gray">
                    Click the play button on steps marked as "Ready" to proceed with your consignment.
                  </Text>
                ) : allSteps.every(s => s.status === 'COMPLETED') && allSteps.length > 0 ? (
                  <Text size="2" style={{ color: '#16a34a' }}>
                    All steps have been completed successfully. Your consignment is ready for processing.
                  </Text>
                ) : (
                  <Text size="2" color="gray">
                    Waiting for dependent steps to be completed before you can proceed.
                  </Text>
                )}
              </Box>
            </Flex>
          </Card>
        </div>
      </div>
    </div>
  )
}