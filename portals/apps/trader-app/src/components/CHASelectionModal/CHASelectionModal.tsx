import { useState, useEffect } from 'react'
import { Dialog, Button, Flex, Text, IconButton, Select } from '@radix-ui/themes'
import { Cross2Icon } from '@radix-ui/react-icons'
import { listChas } from '../../services/consignment'
import { useApi } from '../../services/ApiContext'
import type { CustomsHouseAgent, TradeFlow } from '../../services/types/consignment'

interface CHASelectionModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreate: (chaId: string, flow: TradeFlow) => void
  isCreating?: boolean
}

export function CHASelectionModal({
  open,
  onOpenChange,
  onCreate,
  isCreating = false,
}: CHASelectionModalProps) {
  const api = useApi()
  const [chas, setChas] = useState<CustomsHouseAgent[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedChaId, setSelectedChaId] = useState<string>('')
  const [flow, setFlow] = useState<TradeFlow>('IMPORT')

  useEffect(() => {
    if (!open) return
    setLoading(true)
    listChas(api)
      .then(setChas)
      .catch((err) => {
        console.error('Failed to fetch CHAs:', err)
        setChas([])
      })
      .finally(() => setLoading(false))
  }, [open, api])

  const handleCreate = () => {
    if (selectedChaId) {
      onCreate(selectedChaId, flow)
      onOpenChange(false)
      setSelectedChaId('')
    }
  }

  const handleOpenChange = (isOpen: boolean) => {
    if (!isOpen) setSelectedChaId('')
    onOpenChange(isOpen)
  }

  return (
    <Dialog.Root open={open} onOpenChange={handleOpenChange}>
      <Dialog.Content
        maxWidth="500px"
        onInteractOutside={(e) => e.preventDefault()}
      >
        <Flex justify="between" align="start">
          <Flex direction="column" gap="1">
            <Dialog.Title>New Consignment</Dialog.Title>
            <Dialog.Description size="2" color="gray">
              1st Step: Select a Clearing House Agent. The CHA will then assign the HS Code to start the workflow.
            </Dialog.Description>
          </Flex>
          <Dialog.Close>
            <IconButton variant="ghost" color="gray" size="1">
              <Cross2Icon />
            </IconButton>
          </Dialog.Close>
        </Flex>

        <Flex direction="column" gap="4" style={{ marginTop: 16 }}>
          <Text size="2" weight="bold">Select Trade Flow</Text>
          <Select.Root value={flow} onValueChange={(v) => setFlow(v as TradeFlow)}>
            <Select.Trigger placeholder="Trade flow" />
            <Select.Content>
              <Select.Item value="IMPORT">Import</Select.Item>
              <Select.Item value="EXPORT">Export</Select.Item>
            </Select.Content>
          </Select.Root>

          <Text size="2" weight="bold">Select Clearing House Agent</Text>
          <Select.Root
            value={selectedChaId}
            onValueChange={setSelectedChaId}
            disabled={loading}
          >
            <Select.Trigger placeholder={loading ? 'Loading...' : 'Choose a Major Service Provider...'} />
            <Select.Content>
              {chas.map((cha) => (
                <Select.Item key={cha.id} value={cha.id}>
                  {cha.name}
                </Select.Item>
              ))}
            </Select.Content>
          </Select.Root>

          <Button
            disabled={!selectedChaId || loading || isCreating}
            onClick={handleCreate}
          >
            {isCreating ? 'Creating...' : 'Assign Agent & Create Consignment'}
          </Button>
        </Flex>
      </Dialog.Content>
    </Dialog.Root>
  )
}
