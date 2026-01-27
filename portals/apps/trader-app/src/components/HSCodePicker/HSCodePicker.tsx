import { useState, useEffect } from 'react'
import { Dialog, Button, Box, Flex, Text, Spinner, IconButton, Badge } from '@radix-ui/themes'
import { Cross2Icon, ArrowRightIcon } from '@radix-ui/react-icons'
import { HSCodeSearch } from './HSCodeSearch'
import type {HSCode} from "../../services/types/hsCode.ts";
import type {Workflow} from "../../services/types/workflow.ts";
import {getWorkflowsByHSCode} from "../../services/workflow.ts";

type TradeFlow = 'IMPORT' | 'EXPORT'

interface HSCodePickerProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSelect: (hsCode: HSCode, workflow: Workflow) => void
  /** Whether a consignment is being created */
  isCreating?: boolean
  /** Dialog title */
  title?: string
  /** Confirm button text */
  confirmText?: string
  /** Cancel button text */
  cancelText?: string
}

export function HSCodePicker({
  open,
  onOpenChange,
  onSelect,
  isCreating = false,
  title = 'New Consignment',
  confirmText = 'Start Consignment',
  cancelText = 'Cancel',
}: HSCodePickerProps) {
  const [step, setStep] = useState<'trade-flow' | 'hs-code'>('trade-flow')
  const [tradeFlow, setTradeFlow] = useState<TradeFlow | null>(null)
  const [selectedHSCode, setSelectedHSCode] = useState<HSCode | null>(null)
  const [workflow, setWorkflow] = useState<Workflow | null>(null)
  const [loadingWorkflow, setLoadingWorkflow] = useState(false)
  const [workflowError, setWorkflowError] = useState<string | null>(null)

  useEffect(() => {
    async function fetchWorkflow() {
      if (!selectedHSCode || !tradeFlow) {
        setWorkflow(null)
        setWorkflowError(null)
        return
      }

      setLoadingWorkflow(true)
      setWorkflowError(null)

      try {
        const result = await getWorkflowsByHSCode({ hs_code: selectedHSCode.hsCode })
        const workflows = tradeFlow === 'IMPORT' ? result.import : result.export
        
        if (workflows.length > 0) {
          setWorkflow(workflows[0])
          setWorkflowError(null)
        } else {
          setWorkflow(null)
          setWorkflowError(`No ${tradeFlow.toLowerCase()} workflow available for this HS Code`)
        }
      } catch (error) {
        console.error('Failed to fetch workflow:', error)
        setWorkflow(null)
        setWorkflowError('Failed to load workflow details')
      } finally {
        setLoadingWorkflow(false)
      }
    }

    fetchWorkflow()
  }, [selectedHSCode, tradeFlow])

  const handleConfirm = () => {
    if (selectedHSCode && workflow) {
      onSelect(selectedHSCode, workflow)
      onOpenChange(false)
      resetState()
    }
  }

  const handleTradeFlowSelect = (flow: TradeFlow) => {
    setTradeFlow(flow)
    setStep('hs-code')
  }

  const handleBack = () => {
    if (step === 'hs-code') {
      setStep('trade-flow')
      setSelectedHSCode(null)
      setWorkflow(null)
      setWorkflowError(null)
    }
  }

  const resetState = () => {
    setStep('trade-flow')
    setTradeFlow(null)
    setSelectedHSCode(null)
    setWorkflow(null)
    setWorkflowError(null)
  }

  const handleOpenChange = (isOpen: boolean) => {
    if (!isOpen) {
      resetState()
    }
    onOpenChange(isOpen)
  }

  return (
    <Dialog.Root open={open} onOpenChange={handleOpenChange}>
      <Dialog.Content
        maxWidth="600px"
        style={{ minHeight: '500px', display: 'flex', flexDirection: 'column' }}
        onInteractOutside={(e) => e.preventDefault()}
      >
        <Flex justify="between" align="start">
          <Box>
            <Dialog.Title>{title}</Dialog.Title>
            <Dialog.Description size="2" color="gray">
              {step === 'trade-flow' 
                ? 'Select whether this is an import or export consignment.'
                : `Search and select an HS code for your ${tradeFlow?.toLowerCase()} consignment.`}
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
          {step === 'trade-flow' ? (
            <Flex direction="column" gap="3">
              <Text size="2" weight="medium" color="gray">Select Trade Flow</Text>
              <Flex direction="column" gap="3">
                <button
                  onClick={() => handleTradeFlowSelect('IMPORT')}
                  className="p-6 border-2 border-gray-200 rounded-lg hover:border-blue-500 hover:bg-blue-50 transition-all text-left group cursor-pointer"
                >
                  <Flex align="center" justify="between">
                    <Box>
                      <Text size="4" weight="bold" className="text-gray-900 block mb-1">
                        Import
                      </Text>
                      <Text size="2" color="gray">
                        Bringing goods into the country
                      </Text>
                    </Box>
                    <ArrowRightIcon className="text-gray-400 group-hover:text-blue-500" width="20" height="20" />
                  </Flex>
                </button>
                <button
                  onClick={() => handleTradeFlowSelect('EXPORT')}
                  className="p-6 border-2 border-gray-200 rounded-lg hover:border-green-500 hover:bg-green-50 transition-all text-left group cursor-pointer"
                >
                  <Flex align="center" justify="between">
                    <Box>
                      <Text size="4" weight="bold" className="text-gray-900 block mb-1">
                        Export
                      </Text>
                      <Text size="2" color="gray">
                        Sending goods out of the country
                      </Text>
                    </Box>
                    <ArrowRightIcon className="text-gray-400 group-hover:text-green-500" width="20" height="20" />
                  </Flex>
                </button>
              </Flex>
            </Flex>
          ) : (
            <>
              {/* Step indicator */}
              <Flex align="center" gap="2" mb="4">
                <Badge color={tradeFlow === 'IMPORT' ? 'blue' : 'green'} size="2">
                  {tradeFlow}
                </Badge>
                <Text size="1" color="gray">Selected Trade Flow</Text>
              </Flex>

              {/* HS Code Search */}
              <Box mb="5">
                <HSCodeSearch value={selectedHSCode} onChange={setSelectedHSCode} />
              </Box>

              {/* Workflow Details */}
              {selectedHSCode && (
                <Box>
                  {loadingWorkflow ? (
                    <Flex align="center" justify="center" py="6">
                      <Spinner size="2" />
                      <Text size="2" color="gray" ml="2">
                        Loading workflow details...
                      </Text>
                    </Flex>
                  ) : workflowError ? (
                    <Flex align="center" justify="center" py="6" direction="column" gap="2">
                      <Text size="2" color="red">
                        {workflowError}
                      </Text>
                      <Text size="1" color="gray">
                        Please select a different HS Code
                      </Text>
                    </Flex>
                  ) : workflow ? (
                    <Box p="4" className="bg-blue-50 border border-blue-200 rounded-lg">
                      <Text size="2" weight="bold" className="text-blue-900 block mb-3">
                        Workflow Details
                      </Text>
                      <Flex direction="column" gap="2">
                        <Flex gap="2">
                          <Text size="2" color="gray" style={{ minWidth: '100px' }}>HS Code:</Text>
                          <Text size="2" weight="medium">{selectedHSCode.hsCode}</Text>
                        </Flex>
                        <Flex gap="2">
                          <Text size="2" color="gray" style={{ minWidth: '100px' }}>Description:</Text>
                          <Text size="2" className="text-gray-700" style={{ flex: 1 }}>{selectedHSCode.description}</Text>
                        </Flex>
                        <Flex gap="2">
                          <Text size="2" color="gray" style={{ minWidth: '100px' }}>Trade Flow:</Text>
                          <Text size="2" weight="medium" style={{ textTransform: 'uppercase' }}>
                            {tradeFlow}
                          </Text>
                        </Flex>
                        <Flex gap="2">
                          <Text size="2" color="gray" style={{ minWidth: '100px' }}>Workflow:</Text>
                          <Text size="2" weight="medium">{workflow.name}</Text>
                        </Flex>
                        <Flex gap="2">
                          <Text size="2" color="gray" style={{ minWidth: '100px' }}>Steps:</Text>
                          <Text size="2">{workflow.steps.length} step{workflow.steps.length !== 1 ? 's' : ''}</Text>
                        </Flex>
                      </Flex>
                    </Box>
                  ) : null}
                </Box>
              )}
            </>
          )}
        </Box>

        <Flex gap="3" justify="end" mt="4">
          {step === 'hs-code' && (
            <Button variant="soft" color="gray" onClick={handleBack} disabled={isCreating}>
              Back
            </Button>
          )}
          <Dialog.Close>
            <Button variant="soft" color="gray" disabled={isCreating}>
              {cancelText}
            </Button>
          </Dialog.Close>
          {step === 'hs-code' && (
            <Button
              onClick={handleConfirm}
              disabled={!selectedHSCode || !workflow || loadingWorkflow || isCreating}
              loading={isCreating}
            >
              {isCreating ? 'Creating...' : confirmText}
            </Button>
          )}
        </Flex>
      </Dialog.Content>
    </Dialog.Root>
  )
}