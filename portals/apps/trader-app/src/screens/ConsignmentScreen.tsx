import { useState, useEffect, useRef, type ChangeEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { Badge, Box, Button, Dialog, Flex, IconButton, Select, Spinner, Text, TextField } from '@radix-ui/themes'
import { MagnifyingGlassIcon, PlusIcon } from '@radix-ui/react-icons'
import type { ConsignmentSummary, TradeFlow, ConsignmentState } from "../services/types/consignment.ts"
import { createConsignment, getAllConsignments } from "../services/consignment.ts"
import { useApi } from '../services/ApiContext'
import { getStateColor, formatState, formatDate } from '../utils/consignmentUtils'
import { PaginationControl } from '../components/common/PaginationControl'
import { Cross2Icon, ArrowRightIcon } from '@radix-ui/react-icons'
import { CHASearch, type CHAOption } from '../components/CHAPicker/CHASearch'

// Local alias (avoid a second themes import just for Text-as)
const RadixText = Text

const CHA_OPTIONS = [
  { id: 'a1b2c3d4-0001-4000-8000-000000000001', name: 'Spectra' },
  { id: 'a1b2c3d4-0002-4000-8000-000000000002', name: 'Aitken Spence' },
  { id: 'a1b2c3d4-0003-4000-8000-000000000003', name: 'Advantis' },
  { id: 'a1b2c3d4-0004-4000-8000-000000000004', name: 'Yusen' },
  { id: 'a1b2c3d4-0005-4000-8000-000000000005', name: 'Malship' },
] as const satisfies readonly CHAOption[]

export function ConsignmentScreen() {
  const navigate = useNavigate()
  const api = useApi()
  const [consignments, setConsignments] = useState<ConsignmentSummary[]>([])

  const [totalCount, setTotalCount] = useState(0)
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(0)
  const limit = 50

  // Filters
  const [searchQuery, setSearchQuery] = useState('')
  const [stateFilter, setStateFilter] = useState<string>('all')
  const [tradeFlowFilter, setTradeFlowFilter] = useState<string>('all')

  // Temporary role toggle (until roles come from IDP)
  const [roleView, setRoleView] = useState<'trader' | 'cha'>('trader')
  const [chaId, setChaId] = useState<string>(() => {
    return (
      window.localStorage.getItem('consignments.chaId') ||
      CHA_OPTIONS[2].id // default: Advantis
    )
  })

  // New consignment state
  const [pickerOpen, setPickerOpen] = useState(false)
  const [creating, setCreating] = useState(false)
  const listRequestIdRef = useRef(0)
  const [newStep, setNewStep] = useState<'trade-flow' | 'cha'>('trade-flow')
  const [newFlow, setNewFlow] = useState<TradeFlow | null>(null)
  const [newChaId, setNewChaId] = useState<string>(CHA_OPTIONS[2].id)

  useEffect(() => {
    async function fetchConsignments() {
      const requestId = ++listRequestIdRef.current
      setLoading(true)
      try {
        const data = await getAllConsignments(
          page * limit,
          limit,
          stateFilter as ConsignmentState | 'all',
          tradeFlowFilter as TradeFlow | 'all',
          roleView,
          roleView === 'cha' ? chaId : undefined,
          api
        )
        if (requestId !== listRequestIdRef.current) {
          return
        }
        setConsignments(data.items || [])
        setTotalCount(data.totalCount || 0)
      } catch (error) {
        if (requestId !== listRequestIdRef.current) {
          return
        }
        console.error('Failed to fetch consignments:', error)
      } finally {
        if (requestId === listRequestIdRef.current) {
          setLoading(false)
        }
      }
    }

    fetchConsignments()
  }, [api, page, stateFilter, tradeFlowFilter, roleView, chaId])

  const resetNewConsignment = () => {
    setNewStep('trade-flow')
    setNewFlow(null)
    setNewChaId(CHA_OPTIONS[2].id)
  }

  const handleNewOpenChange = (open: boolean) => {
    setPickerOpen(open)
    if (!open) resetNewConsignment()
  }

  const handleCreateShell = async () => {
    if (!newFlow) return
    setCreating(true)
    try {
      const response = await createConsignment(
        {
          flow: newFlow,
          chaId: newChaId,
        },
        api
      )
      // After creating, make CHA view filter match the new consignment's chaId
      setChaId(newChaId)
      window.localStorage.setItem('consignments.chaId', newChaId)
      handleNewOpenChange(false)
      // ensure list refreshes (user will see it in both views)
      navigate(`/consignments/${response.id}`)
    } catch (error) {
      console.error('Failed to create consignment:', error)
    } finally {
      setCreating(false)
    }
  }

  const filteredConsignments = consignments.filter((c) => {
    const item = c.items?.[0]
    const hsCode = item?.hsCode?.hsCode || ''
    const description = item?.hsCode?.description || ''
    const matchesSearch =
      searchQuery === '' ||
      c.id.toLowerCase().includes(searchQuery.toLowerCase()) ||
      hsCode.toLowerCase().includes(searchQuery.toLowerCase()) ||
      description.toLowerCase().includes(searchQuery.toLowerCase())

    return matchesSearch
  })



  if (loading) {
    return (
      <div className="p-6">
        <div className="flex items-center justify-center py-12">
          <Spinner size="3" />
          <Text size="3" color="gray" className="ml-3">
            Loading Consignments...
          </Text>
        </div>
      </div>
    )
  }

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">Consignments</h1>
        <div className="flex gap-2">
          <Select.Root
            value={roleView}
            onValueChange={(val: string) => {
              const next = val === 'cha' ? 'cha' : 'trader'
              setRoleView(next)
              setPage(0)
              setPickerOpen(false)
            }}
          >
            <Select.Trigger placeholder="Role View" />
            <Select.Content>
              <Select.Item value="trader">Trader</Select.Item>
              <Select.Item value="cha">CHA</Select.Item>
            </Select.Content>
          </Select.Root>

          {roleView === 'cha' ? null : (
            <Button onClick={() => handleNewOpenChange(true)} disabled={creating}>
              <PlusIcon />
              {creating ? 'Creating...' : 'New Consignment'}
            </Button>
          )}
        </div>
      </div>

      {/* New consignment (two-stage): Flow -> CHA */}
      <Dialog.Root open={pickerOpen} onOpenChange={handleNewOpenChange}>
        <Dialog.Content
          maxWidth="600px"
          style={{ minHeight: '420px', display: 'flex', flexDirection: 'column' }}
          onInteractOutside={(e: Event) => e.preventDefault()}
        >
          <Flex justify="between" align="start">
            <Box>
              <Dialog.Title>New Consignment</Dialog.Title>
              <Dialog.Description size="2" color="gray">
                {newStep === 'trade-flow'
                  ? 'Select whether this is an import or export consignment.'
                  : 'Select the Customs House Agent (CHA) for this consignment.'}
              </Dialog.Description>
            </Box>
            <Dialog.Close>
              <IconButton variant="ghost" color="gray" size="1">
                <Cross2Icon />
              </IconButton>
            </Dialog.Close>
          </Flex>

          <Box mt="4" />

          <Box style={{ flex: 1 }}>
            {newStep === 'trade-flow' ? (
              <Flex direction="column" gap="3">
                <RadixText size="2" weight="medium" color="gray">
                  Select Trade Flow
                </RadixText>
                <Flex direction="column" gap="3">
                  <button
                    onClick={() => {
                      setNewFlow('IMPORT')
                      setNewStep('cha')
                    }}
                    className="p-6 border-2 border-gray-200 rounded-lg hover:border-blue-500 hover:bg-blue-50 transition-all text-left group cursor-pointer"
                  >
                    <Flex align="center" justify="between">
                      <Box>
                        <RadixText size="4" weight="bold" className="text-gray-900 block mb-1">
                          Import
                        </RadixText>
                        <RadixText size="2" color="gray">
                          Bringing goods into the country
                        </RadixText>
                      </Box>
                      <ArrowRightIcon className="text-gray-400 group-hover:text-blue-500" width="20" height="20" />
                    </Flex>
                  </button>
                  <button
                    onClick={() => {
                      setNewFlow('EXPORT')
                      setNewStep('cha')
                    }}
                    className="p-6 border-2 border-gray-200 rounded-lg hover:border-green-500 hover:bg-green-50 transition-all text-left group cursor-pointer"
                  >
                    <Flex align="center" justify="between">
                      <Box>
                        <RadixText size="4" weight="bold" className="text-gray-900 block mb-1">
                          Export
                        </RadixText>
                        <RadixText size="2" color="gray">
                          Sending goods out of the country
                        </RadixText>
                      </Box>
                      <ArrowRightIcon className="text-gray-400 group-hover:text-green-500" width="20" height="20" />
                    </Flex>
                  </button>
                </Flex>
              </Flex>
            ) : (
              <Flex direction="column" gap="3">
                <RadixText size="2" weight="medium" color="gray">
                  Select CHA
                </RadixText>
                <CHASearch
                  options={CHA_OPTIONS}
                  value={CHA_OPTIONS.find((c) => c.id === newChaId) ?? null}
                  onChange={(cha) => {
                    if (cha) setNewChaId(cha.id)
                  }}
                />
                <RadixText size="1" color="gray">
                  Flow: {newFlow}
                </RadixText>
              </Flex>
            )}
          </Box>

          <Flex gap="3" justify="end" mt="4">
            {newStep === 'cha' ? (
              <Button
                variant="soft"
                color="gray"
                onClick={() => {
                  setNewStep('trade-flow')
                  setNewFlow(null)
                }}
                disabled={creating}
              >
                Back
              </Button>
            ) : null}
            <Dialog.Close>
              <Button variant="soft" color="gray" disabled={creating}>
                Cancel
              </Button>
            </Dialog.Close>
            {newStep === 'cha' ? (
              <Button onClick={handleCreateShell} disabled={!newFlow || creating} loading={creating}>
                {creating ? 'Creating...' : 'Create Consignment'}
              </Button>
            ) : null}
          </Flex>
        </Dialog.Content>
      </Dialog.Root>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
        <div className="bg-white rounded-lg shadow p-6">
          <h3 className="text-sm font-medium text-gray-500">Total Consignments</h3>
          <p className="mt-2 text-3xl font-semibold text-gray-900">{totalCount}</p>
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
                onChange={(e: ChangeEvent<HTMLInputElement>) => setSearchQuery(e.target.value)}
              >
                <TextField.Slot>
                  <MagnifyingGlassIcon height="16" width="16" />
                </TextField.Slot>
              </TextField.Root>
            </div>
            <div className="flex gap-3">
              {roleView === 'cha' ? (
                <Select.Root
                  value={chaId}
                  onValueChange={(val: string) => {
                    setChaId(val)
                    window.localStorage.setItem('consignments.chaId', val)
                    setPage(0)
                  }}
                >
                  <Select.Trigger placeholder="CHA" />
                  <Select.Content>
                    {CHA_OPTIONS.map((c) => (
                      <Select.Item key={c.id} value={c.id}>
                        {c.name}
                      </Select.Item>
                    ))}
                  </Select.Content>
                </Select.Root>
              ) : null}
              <Select.Root
                value={stateFilter}
                onValueChange={(val: string) => {
                  setStateFilter(val)
                  setPage(0)
                }}
              >
                <Select.Trigger placeholder="State" />
                <Select.Content>
                  <Select.Item value="all">All States</Select.Item>
                  <Select.Item value="IN_PROGRESS">In Progress</Select.Item>
                  <Select.Item value="FINISHED">Finished</Select.Item>
                  <Select.Item value="REQUIRES_REWORK">Requires Rework</Select.Item>
                </Select.Content>
              </Select.Root>
              <Select.Root
                value={tradeFlowFilter}
                onValueChange={(val: string) => {
                  setTradeFlowFilter(val)
                  setPage(0)
                }}
              >
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
                ? roleView === 'cha'
                  ? 'No consignments yet.'
                  : 'No consignments yet. Click "New Consignment" to create your first one.'
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
                  {/* HS Code Column removed as per request */}
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
                  const completedSteps = consignment.completedWorkflowNodeCount || 0
                  const totalSteps = consignment.workflowNodeCount || 0

                  return (
                    <tr
                      key={consignment.id}
                      onClick={() => {
                        if (roleView === 'cha') {
                          navigate(`/consignments/${consignment.id}?view=cha&cha_id=${encodeURIComponent(chaId)}`)
                        } else {
                          navigate(`/consignments/${consignment.id}`)
                        }
                      }}
                      className="hover:bg-gray-50 cursor-pointer transition-colors"
                    >
                      <td className="px-6 py-4 whitespace-nowrap">
                        <Text size="2" weight="medium" className="text-blue-600 font-mono">
                          {consignment.id}
                        </Text>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <Badge
                          size="1"
                          color={consignment.flow === 'IMPORT' ? 'blue' : 'green'}
                          variant="soft"
                        >
                          {consignment.flow}
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
                          {consignment.createdAt ? formatDate(consignment.createdAt) : '-'}
                        </Text>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
        <PaginationControl
          currentPage={page + 1}
          totalPages={Math.ceil(totalCount / limit)}
          onPageChange={(p) => setPage(p - 1)}
          hasNext={(page + 1) * limit < totalCount}
          hasPrev={page > 0}
          totalCount={totalCount}

        />
      </div>
    </div >
  )
}