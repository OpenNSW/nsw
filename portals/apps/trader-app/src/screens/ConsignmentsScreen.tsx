import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Badge, Text, TextField, Spinner, Select, Button, AlertDialog, Box, Flex } from '@radix-ui/themes'
import { MagnifyingGlassIcon, PlusIcon } from '@radix-ui/react-icons'
import { HSCodePicker } from '../components/HSCodePicker'
import { createConsignment, getAllConsignments } from "../services/consignment.ts"
import type { Workflow } from "../services/types/workflow.ts"
import type { HSCode } from "../services/types/hsCode.ts"
import type { Consignment, ConsignmentState, TradeFlow } from "../services/types/consignment.ts"

function getStateColor(state: ConsignmentState): 'orange' | 'green' | 'gray' {
  switch (state) {
    case 'IN_PROGRESS':
      return 'orange'
    case 'COMPLETED':
      return 'green'
    case 'CANCELLED':
      return 'gray'
    default:
      return 'gray'
  }
}

function formatState(state: ConsignmentState): string {
  return state.replace('_', ' ').replace(/\b\w/g, (c) => c.toUpperCase())
}

function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

export function ConsignmentsScreen() {
  const navigate = useNavigate()
  const [consignments, setConsignments] = useState<Consignment[]>([])
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [stateFilter, setStateFilter] = useState<string>('all')
  const [tradeFlowFilter, setTradeFlowFilter] = useState<string>('all')

  // New consignment state
  const [pickerOpen, setPickerOpen] = useState(false)
  const [creating, setCreating] = useState(false)
  const [pendingSelection, setPendingSelection] = useState<{ hsCode: HSCode; workflow: Workflow } | null>(null)

  useEffect(() => {
    async function fetchConsignments() {
      try {
        const data = await getAllConsignments()
        setConsignments(data)
      } catch (error) {
        console.error('Failed to fetch consignments:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchConsignments()
  }, [])

  const filteredConsignments = consignments.filter((c) => {
    const hsCodeId = c.items[0]?.hsCodeID || ''
    const matchesSearch =
      searchQuery === '' ||
      c.id.toLowerCase().includes(searchQuery.toLowerCase()) ||
      hsCodeId.toLowerCase().includes(searchQuery.toLowerCase())

    const matchesState = stateFilter === 'all' || c.state === stateFilter
    const matchesTradeFlow = tradeFlowFilter === 'all' || c.tradeFlow === tradeFlowFilter

    return matchesSearch && matchesState && matchesTradeFlow
  })

  const handleSelect = (hsCode: HSCode, workflow: Workflow) => {
    setPickerOpen(false)
    setPendingSelection({ hsCode, workflow })
  }

  const handleConfirmCreate = async () => {
    if (!pendingSelection) return

    const { hsCode, workflow } = pendingSelection
    setCreating(true)
    setPendingSelection(null)

    try {
      const response = await createConsignment({
        tradeFlow: workflow.type.toUpperCase() as TradeFlow,
        traderId: 'trader-123', // TODO: Get from auth context
        items: [
          {
            hsCodeId: hsCode.id,
            metadata: {},
            workflowTemplateId: workflow.id,
          },
        ],
      })

      navigate(`/consignments/${response.id}`)
    } catch (error) {
      console.error('Failed to create consignment:', error)
    } finally {
      setCreating(false)
    }
  }

  if (loading) {
    return (
      <div className="p-6">
        <div className="flex items-center justify-center py-12">
          <Spinner size="3" />
          <Text size="3" color="gray" className="ml-3">
            Loading consignments...
          </Text>
        </div>
      </div>
    )
  }

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">Consignments</h1>
        <div className="flex items-center gap-4">
          <Text size="2" color="gray">
            {filteredConsignments.length} of {consignments.length} consignments
          </Text>
          <Button onClick={() => setPickerOpen(true)} disabled={creating}>
            <PlusIcon />
            {creating ? 'Creating...' : 'New Consignment'}
          </Button>
        </div>
      </div>

      <div className="bg-white rounded-lg shadow mb-6">
        <div className="p-4 border-b border-gray-200">
          <div className="flex flex-col md:flex-row gap-4">
            <div className="flex-1">
              <TextField.Root
                size="2"
                placeholder="Search by ID or HS Code..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              >
                <TextField.Slot>
                  <MagnifyingGlassIcon height="16" width="16" />
                </TextField.Slot>
              </TextField.Root>
            </div>
            <div className="flex gap-3">
              <Select.Root value={stateFilter} onValueChange={setStateFilter}>
                <Select.Trigger placeholder="State" />
                <Select.Content>
                  <Select.Item value="all">All States</Select.Item>
                  <Select.Item value="IN_PROGRESS">In Progress</Select.Item>
                  <Select.Item value="COMPLETED">Completed</Select.Item>
                  <Select.Item value="CANCELLED">Cancelled</Select.Item>
                </Select.Content>
              </Select.Root>
              <Select.Root value={tradeFlowFilter} onValueChange={setTradeFlowFilter}>
                <Select.Trigger placeholder="Trade Flow" />
                <Select.Content>
                  <Select.Item value="all">All Types</Select.Item>
                  <Select.Item value="IMPORT">Import</Select.Item>
                  <Select.Item value="EXPORT">Export</Select.Item>
                </Select.Content>
              </Select.Root>
            </div>
          </div>
        </div>

        {filteredConsignments.length === 0 ? (
          <div className="p-12 text-center">
            <Text size="3" color="gray">
              {consignments.length === 0
                ? 'No consignments yet. Click "New Consignment" to create your first one.'
                : 'No consignments match your filters.'}
            </Text>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-gray-200 bg-gray-50">
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Consignment ID
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    HS Code
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Trade Flow
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    State
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Steps
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Created
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {filteredConsignments.map((consignment) => {
                  const item = consignment.items[0]
                  const completedSteps = item?.steps.filter(s => s.status === 'COMPLETED').length || 0
                  const totalSteps = item?.steps.length || 0

                  return (
                    <tr
                      key={consignment.id}
                      onClick={() => navigate(`/consignments/${consignment.id}`)}
                      className="hover:bg-gray-50 cursor-pointer transition-colors"
                    >
                      <td className="px-6 py-4 whitespace-nowrap">
                        <Text size="2" weight="medium" className="text-blue-600">
                          {consignment.id.slice(0, 8)}...
                        </Text>
                      </td>
                      <td className="px-6 py-4">
                        <Text size="2" weight="medium">
                          {item?.hsCodeID || '-'}
                        </Text>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <Badge
                          size="1"
                          color={consignment.tradeFlow === 'IMPORT' ? 'blue' : 'green'}
                          variant="soft"
                        >
                          {consignment.tradeFlow}
                        </Badge>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <Badge size="1" color={getStateColor(consignment.state)}>
                          {formatState(consignment.state)}
                        </Badge>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <Text size="2" color="gray">
                          {completedSteps}/{totalSteps}
                        </Text>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <Text size="2" color="gray">
                          {formatDate(consignment.createdAt)}
                        </Text>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <HSCodePicker
        open={pickerOpen}
        onOpenChange={setPickerOpen}
        onSelect={handleSelect}
        title="New Consignment"
        description="Select an HS Code and workflow to create a new consignment."
        confirmText="Continue"
      />

      {/* Confirmation Dialog */}
      <AlertDialog.Root open={!!pendingSelection} onOpenChange={(open) => !open && setPendingSelection(null)}>
        <AlertDialog.Content maxWidth="450px">
          <AlertDialog.Title>Start Consignment</AlertDialog.Title>
          <AlertDialog.Description size="2">
            Are you sure you want to start a new consignment with the following details?
          </AlertDialog.Description>

          {pendingSelection && (
            <Box mt="4" p="3" className="bg-gray-50 rounded-md">
              <Flex direction="column" gap="2">
                <Flex justify="between">
                  <Text size="2" color="gray">HS Code:</Text>
                  <Text size="2" weight="medium">{pendingSelection.hsCode.hsCode}</Text>
                </Flex>
                <Flex justify="between">
                  <Text size="2" color="gray">Description:</Text>
                  <Text size="2" weight="medium" style={{ textAlign: 'right', maxWidth: '250px' }}>
                    {pendingSelection.hsCode.description}
                  </Text>
                </Flex>
                <Flex justify="between">
                  <Text size="2" color="gray">Workflow:</Text>
                  <Text size="2" weight="medium">{pendingSelection.workflow.name}</Text>
                </Flex>
                <Flex justify="between">
                  <Text size="2" color="gray">Trade Flow:</Text>
                  <Text size="2" weight="medium" style={{ textTransform: 'uppercase' }}>
                    {pendingSelection.workflow.type}
                  </Text>
                </Flex>
              </Flex>
            </Box>
          )}

          <Flex gap="3" mt="4" justify="end">
            <AlertDialog.Cancel>
              <Button variant="soft" color="gray">
                Cancel
              </Button>
            </AlertDialog.Cancel>
            <AlertDialog.Action>
              <Button onClick={handleConfirmCreate}>
                Start Consignment
              </Button>
            </AlertDialog.Action>
          </Flex>
        </AlertDialog.Content>
      </AlertDialog.Root>
    </div>
  )
}