import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button } from '@radix-ui/themes'
import { HSCodePicker } from '../components/HSCodePicker'
import type { HSCode } from "../services/types/hsCode.ts"
import type { Workflow } from "../services/types/workflow.ts"
import type { Consignment, TradeFlow } from "../services/types/consignment.ts"
import { createConsignment, getAllConsignments } from "../services/consignment.ts"

export function DashboardScreen() {
  const navigate = useNavigate()
  const [pickerOpen, setPickerOpen] = useState(false)
  const [creating, setCreating] = useState(false)

  const [consignments, setConsignments] = useState<Consignment[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    async function fetchConsignments() {
      try {
        const data = await getAllConsignments()
        setConsignments(data.items)
      } catch (error) {
        console.error('Failed to fetch consignments:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchConsignments()
  }, [])

  const handleSelect = async (hsCode: HSCode, workflow: Workflow) => {
    setCreating(true)

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

      setPickerOpen(false)
      // Navigate to the consignment detail page
      navigate(`/consignments/${response.id}`)
    } catch (error) {
      console.error('Failed to create consignment:', error)
      // Could show an error toast here
    } finally {
      setCreating(false)
    }
  }

  const totalConsignments = consignments.length
  const inProgressConsignments = consignments.filter(c => c.state === 'IN_PROGRESS').length
  const completedConsignments = consignments.filter(c => c.state === 'FINISHED').length

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">Dashboard</h1>
        <Button onClick={() => setPickerOpen(true)} disabled={creating}>
          {creating ? 'Creating...' : 'New Consignment'}
        </Button>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-white rounded-lg shadow p-6">
          <h3 className="text-sm font-medium text-gray-500">Total Consignments</h3>
          <p className="mt-2 text-3xl font-semibold text-gray-900">{loading ? '-' : totalConsignments}</p>
        </div>
        <div className="bg-white rounded-lg shadow p-6">
          <h3 className="text-sm font-medium text-gray-500">In Progress</h3>
          <p className="mt-2 text-3xl font-semibold text-gray-900">{loading ? '-' : inProgressConsignments}</p>
        </div>
        <div className="bg-white rounded-lg shadow p-6">
          <h3 className="text-sm font-medium text-gray-500">Completed</h3>
          <p className="mt-2 text-3xl font-semibold text-gray-900">{loading ? '-' : completedConsignments}</p>
        </div>
      </div>

      <HSCodePicker
        open={pickerOpen}
        onOpenChange={setPickerOpen}
        onSelect={handleSelect}
        isCreating={creating}
      />
    </div>
  )
}