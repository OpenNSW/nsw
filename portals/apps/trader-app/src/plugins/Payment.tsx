import { useState } from "react"
import { useParams } from "react-router-dom"
import { Box, Button, Dialog, Flex, IconButton, Text } from "@radix-ui/themes"
import { Cross2Icon } from "@radix-ui/react-icons"
import { useApi } from "../services/ApiContext"
import { getTaskInfo, sendTaskAction, type TaskCommandResponse } from "../services/task"

type BreakDown = {
  description: string
  category: string
  type: string
  quantity: number
  unitPrice: number
  amount: number
}

export type PaymentConfigs = {
  gatewayUrl: string
  totalAmount: number
  currency: string
  breakdown: BreakDown[];
}

export default function Payment(props: {
  configs: PaymentConfigs
  pluginState: string
  onTaskUpdated?: () => Promise<void>
}) {
  const { consignmentId, preConsignmentId, taskId } = useParams<{
    consignmentId?: string
    preConsignmentId?: string
    taskId?: string
  }>()

  const [isInitiating, setIsInitiating] = useState(false)
  const [isProcessingResult, setIsProcessingResult] = useState(false)
  const [isPopupOpen, setIsPopupOpen] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)
  const api = useApi()

  const workflowId = preConsignmentId || consignmentId
  const isCompleted = props.pluginState === "COMPLETED"
  const gatewayUrl = props.configs?.gatewayUrl ?? ""

  const refreshGatewaySession = async () => {
    if (!taskId) {
      return
    }

    try {
      await getTaskInfo(taskId, api)
    } catch (err) {
      console.error("Error refreshing payment session:", err)
    }
  }

  const isSessionExpiredResponse = (response: TaskCommandResponse): boolean => {
    if (response.success) {
      return false
    }

    const code = response.error?.code ?? ""
    const message = response.error?.message?.toLowerCase() ?? ""
    return code === "SESSION_EXPIRED" || message.includes("session expired")
  }

  const isSessionExpiredError = (err: unknown): boolean => {
    const message = err instanceof Error ? err.message : String(err)
    return message.toLowerCase().includes("session expired")
  }

  const handlePayNow = async () => {
    if (!workflowId || !taskId) {
      setSubmitError("Workflow ID or Task ID is missing.")
      return
    }

    if (props.pluginState === "IN_PROGRESS") {
      if (!gatewayUrl) {
        setSubmitError("Gateway URL is not available.")
        return
      }

      setSubmitError(null)
      setIsPopupOpen(true)
      return
    }

    const initiatePayment = async (allowRetry: boolean): Promise<boolean> => {
      try {
        const response = await sendTaskAction(taskId, workflowId, "INITIATE_PAYMENT")
        if (response.success) {
          return true
        }

        if (allowRetry && isSessionExpiredResponse(response)) {
          await refreshGatewaySession()
          return initiatePayment(false)
        }

        setSubmitError(response.error?.message ?? "Failed to initiate payment.")
        return false
      } catch (err) {
        if (allowRetry && isSessionExpiredError(err)) {
          await refreshGatewaySession()
          return initiatePayment(false)
        }

        console.error("Error initiating payment:", err)
        setSubmitError("Failed to initiate payment. Please try again.")
        return false
      }
    }

    setIsInitiating(true)
    setSubmitError(null)

    try {
      const initiated = await initiatePayment(true)
      if (!initiated) {
        return
      }

      setIsPopupOpen(true)
    } finally {
      setIsInitiating(false)
    }
  }

  const handleMockGatewayResult = async (action: "PAYMENT_SUCCESS" | "PAYMENT_FAILED") => {
    if (!workflowId || !taskId) {
      setSubmitError("Workflow ID or Task ID is missing.")
      return
    }

    setIsProcessingResult(true)
    setSubmitError(null)

    try {
      const response = await sendTaskAction(taskId, workflowId, action)
      if (!response.success) {
        setSubmitError(response.error?.message ?? "Failed to process payment result.")
        return
      }

      setIsPopupOpen(false)

      if (props.onTaskUpdated) {
        await props.onTaskUpdated()
      }
    } catch (err) {
      console.error("Error processing payment result:", err)
      setSubmitError("Failed to process payment result. Please try again.")
    } finally {
      setIsProcessingResult(false)
    }
  }

  return (
    <div className="bg-white rounded-lg shadow-md p-6 space-y-4">
      {/* Payment Breakdown Table */}
      <Box className="border border-gray-200 rounded-lg overflow-hidden">
        <div className="bg-gray-50 px-4 py-2 border-b border-gray-200">
          <Text size="2" weight="bold" color="gray">Payment Breakdown</Text>
        </div>
        <div className="p-0">
          <table className="w-full text-left text-sm">
            <thead>
            <tr className="border-b border-gray-100 bg-gray-50/50">
              <th className="px-4 py-2 font-semibold text-gray-600">Description</th>
              <th className="px-4 py-2 font-semibold text-gray-600 text-right">Details</th>
              <th className="px-4 py-2 font-semibold text-gray-600 text-right">Amount ({props.configs.currency})</th>
            </tr>
            </thead>
            <tbody className="divide-y divide-gray-50">
            {props.configs.breakdown.map((item, idx) => (
              <tr key={idx} className="hover:bg-gray-50/30 transition-colors">
                <td className="px-4 py-3 text-gray-800 font-medium">{item.description}</td>
                <td className="px-4 py-3 text-right text-gray-500 text-xs">
                  {item.type === "FIXED" ? (
                    <>{item.quantity} × {item.unitPrice.toLocaleString(undefined, { minimumFractionDigits: 2 })}</>
                  ) : (
                    <span className="italic">Calculated based on running total</span>
                  )}
                </td>
                <td className={`px-4 py-3 text-right font-mono ${item.category === 'DEDUCTION' ? 'text-red-600' : 'text-gray-900'}`}>
                  {item.category === 'DEDUCTION' ? '-' : ''}
                  {item.amount.toLocaleString(undefined, { minimumFractionDigits: 2 })}
                </td>
              </tr>
            ))}
            </tbody>
            <tfoot>
            <tr className="bg-blue-50/50">
              <td colSpan={2} className="px-4 py-4 text-right font-bold text-blue-900 uppercase tracking-wider">Total Payable</td>
              <td className="px-4 py-4 text-right text-lg font-black text-blue-900 font-mono">
                {props.configs.totalAmount.toLocaleString(undefined, { minimumFractionDigits: 2 })} {props.configs.currency}
              </td>
            </tr>
            </tfoot>
          </table>
        </div>
      </Box>
      <h1 className="text-2xl font-bold text-gray-800">Payment</h1>

      <div className="text-sm text-gray-700">
  		{isCompleted ? "Paid Amount" : "Amount"}: <span className="font-medium">{props.configs.totalAmount} {props.configs.currency}</span>
      </div>

      {!isCompleted && (
        <Button onClick={() => { void handlePayNow() }} disabled={isInitiating} size="3">
          {isInitiating ? "Initiating..." : "Pay Now"}
        </Button>
      )}

      <Dialog.Root open={isPopupOpen} onOpenChange={setIsPopupOpen}>
        <Dialog.Content maxWidth="520px">
          <Flex justify="between" align="start">
            <Box>
              <Dialog.Title>Mock Payment Gateway</Dialog.Title>
              <Dialog.Description size="2" color="gray">
                Mock popup for testing payment result actions.
              </Dialog.Description>
            </Box>
            <Dialog.Close>
              <IconButton variant="ghost" color="gray" size="1">
                <Cross2Icon />
              </IconButton>
            </Dialog.Close>
          </Flex>

          <Box mt="4" className="space-y-3">
            <div className="bg-gray-50 border border-gray-200 rounded-lg p-4">
              <Text size="2" color="gray">Popup URL</Text>
              <p className="text-sm text-gray-900 break-all mt-1">{gatewayUrl}</p>
            </div>

            <div className="bg-gray-50 border border-gray-200 rounded-lg p-4">
              <Text size="2" color="gray">Amount</Text>
              <p className="text-sm text-gray-900 mt-1">{props.configs.totalAmount} {props.configs.currency}</p>
            </div>
          </Box>

          <Flex gap="3" justify="end" mt="5">
            <Button
              variant="soft"
              color="red"
              disabled={isProcessingResult}
              onClick={() => { void handleMockGatewayResult("PAYMENT_FAILED") }}
            >
              {isProcessingResult ? "Processing..." : "Mark Failed"}
            </Button>
            <Button
              color="green"
              disabled={isProcessingResult}
              onClick={() => { void handleMockGatewayResult("PAYMENT_SUCCESS") }}
            >
              {isProcessingResult ? "Processing..." : "Mark Success"}
            </Button>
          </Flex>
        </Dialog.Content>
      </Dialog.Root>

      {submitError && (
        <div className="bg-red-100 text-red-700 rounded-lg p-4">
          <p>{submitError}</p>
        </div>
      )}
    </div>
  )
}