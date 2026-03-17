import { useEffect, useState } from "react"
import { useParams } from "react-router-dom"
import { Box, Button, Dialog, Flex, IconButton, Text } from "@radix-ui/themes"
import { Cross2Icon } from "@radix-ui/react-icons"
import { sendTaskAction } from "../services/task"

export type PaymentConfigs = {
  gatewayUrl: string
  amount: number
  currency: string
}

export default function Payment(props: {
  configs: PaymentConfigs | null
  pluginState: string
  onTaskUpdated?: () => Promise<void>
}) {
  const { consignmentId, preConsignmentId, taskId } = useParams<{
    consignmentId?: string; preConsignmentId?: string; taskId?: string
  }>()

  const [isInitiating, setIsInitiating] = useState(false)
  const [isPopupOpen, setIsPopupOpen] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)
  
  // Check for errors returned via query params after failed mock/real gateway
  const urlParams = new URLSearchParams(window.location.search);
  const paymentError = urlParams.has('payment_error');

  useEffect(() => {
    if (paymentError) {
      setSubmitError("Payment failed or timed out.");
      window.history.replaceState({}, document.title, window.location.pathname);
    }
  }, [paymentError]);

  const workflowId = preConsignmentId || consignmentId
  const isCompleted = props.pluginState === "COMPLETED"
  const amount = props.configs?.amount ?? 0
  const currency = props.configs?.currency ?? ""

  // --- INITIATE LOGIC ---
  const handlePayNow = async (method: 'CARD') => {
    if (!workflowId || !taskId) return;
    setIsInitiating(true);
    setSubmitError(null);

    try {
      // Construct the exact URL the gateway should send the user back to
      const returnUrl = `${window.location.origin}/payment-return?taskId=${taskId}&workflowId=${workflowId}`;

      // Pass this returnUrl to the backend
      const response = await sendTaskAction(taskId, workflowId, "INITIATE_PAYMENT", {
        method,
        returnUrl
      });

      if (response.success) {
        const nextUrl = (response.data as any)?.gatewayUrl;
        if (nextUrl) {
          // Redirect to external gateway
          window.location.href = nextUrl;
        } else {
          // If no URL, check if we need to show the mock terminal or if it succeeded instantly
          // We need to fetch the latest state or check the response data
          const isNowCompleted = (response.data as any)?.message?.toLowerCase().includes("completed") || (response.data as any)?.status === "COMPLETED";

          if (isNowCompleted) {
            if (props.onTaskUpdated) await props.onTaskUpdated();
            else window.location.reload();
          } else {
            // Still IN_PROGRESS, so let the user simulate the callback manually via the popup
            setIsPopupOpen(true);
          }
        }
      } else {
        setSubmitError(response.error?.message ?? "Failed to initiate payment.");
      }
    } catch (err) {
      setSubmitError("Failed to initiate payment. Please try again.");
    } finally {
      setIsInitiating(false);
    }
  }

  const handleMockGatewayResult = async (action: "PAYMENT_SUCCESS" | "PAYMENT_FAILED") => {
    if (!workflowId || !taskId) return;
    setIsInitiating(true);
    setSubmitError(null);

    try {
      const response = await sendTaskAction(taskId, workflowId, action);
      if (response.success) {
        setIsPopupOpen(false);
        if (props.onTaskUpdated) await props.onTaskUpdated();
      } else {
        setSubmitError(response.error?.message ?? "Failed to process mock payment.");
      }
    } catch (err) {
      setSubmitError("Failed to process mock payment. Please try again.");
    } finally {
      setIsInitiating(false);
    }
  }

  return (
    <div className="bg-white rounded-lg shadow-md p-6 space-y-4">
      <h1 className="text-2xl font-bold text-gray-800">Payment</h1>

      <div className="text-sm text-gray-700">
        {isCompleted ? "Paid Amount" : "Amount"}: <span className="font-medium">{amount} {currency}</span>
      </div>

      {!isCompleted && (
        <Flex gap="3">
          <Button
            onClick={() => { void handlePayNow('CARD') }}
            disabled={isInitiating || props.pluginState === "COMPLETED"}
            size="3"
            color="blue"
          >
            {isInitiating ? "Initiating..." : "Pay Now"}
          </Button>
        </Flex>
      )}

      {isCompleted && (
        <div className="bg-green-100 text-green-700 rounded-lg p-4">
          <p className="font-medium">Payment Successful!</p>
          <p className="text-sm">Your transaction has been processed and confirmed.</p>
        </div>
      )}

      <Dialog.Root open={isPopupOpen} onOpenChange={setIsPopupOpen}>
        <Dialog.Content maxWidth="520px">
          <Flex justify="between" align="start">
            <Box>
              <Dialog.Title>Mock Payment Gateway</Dialog.Title>
            </Box>
            <Dialog.Close>
              <IconButton variant="ghost" color="gray" size="1">
                <Cross2Icon />
              </IconButton>
            </Dialog.Close>
          </Flex>

          <Box mt="4" className="space-y-3">
            <div className="bg-gray-50 border border-gray-200 rounded-lg p-4">
              <Text size="2" color="gray">Simulation Mode</Text>
              <p className="text-sm text-gray-900 mt-1">
                You are currently using the mock payment gateway. Choose an outcome to simulate the external provider response.
              </p>
            </div>

            <div className="bg-gray-50 border border-gray-200 rounded-lg p-4">
              <Text size="2" color="gray">Amount</Text>
              <p className="text-sm text-gray-900 mt-1">{amount} {currency}</p>
            </div>
          </Box>

          <Flex gap="3" justify="end" mt="5">
            <Button
              variant="soft"
              color="red"
              disabled={isInitiating}
              onClick={() => { void handleMockGatewayResult("PAYMENT_FAILED") }}
            >
              {isInitiating ? "Processing..." : "Simulate Failure"}
            </Button>
            <Button
              color="green"
              disabled={isInitiating}
              onClick={() => { void handleMockGatewayResult("PAYMENT_SUCCESS") }}
            >
              {isInitiating ? "Processing..." : "Simulate Success"}
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
  );
}