import { useEffect, useState } from "react"
import { useSearchParams, useNavigate } from "react-router-dom"
import { useApi } from "../services/ApiContext"
import { getTaskInfo } from "../services/task"

export function PaymentReturnScreen() {
  const [searchParams] = useSearchParams()
  const taskId = searchParams.get('taskId')
  const workflowId = searchParams.get('workflowId')
  const navigate = useNavigate()
  const api = useApi()
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!taskId || !workflowId) {
      setError("Missing task or workflow information.")
      return
    }

    let intervalId: any = setInterval(async () => {
      try {
        const updatedInfo = await getTaskInfo(taskId, api);
        if (updatedInfo.pluginState === "COMPLETED") {
          clearInterval(intervalId);
          // Navigate back to the task, could be standalone consignment or preconsignment
          // Assuming generic fallback link if type is unknown, or we can check the path structure
          // `workflowId` is typically consignmentId. Since we don't know if it's pre / standard, we just
          // use consignments. The user can navigate manually or we can provide a safe back-link.
          navigate(`/consignments/${workflowId}/tasks/${taskId}`);
        } else if (updatedInfo.pluginState === "IDLE") {
          clearInterval(intervalId);
          navigate(`/consignments/${workflowId}/tasks/${taskId}?payment_error=true`);
        }
      } catch (err) {
        console.error("Error polling task status:", err);
      }
    }, 2000);

    return () => clearInterval(intervalId);
  }, [taskId, workflowId, api, navigate]);

  if (error) {
    return (
      <div className="bg-white rounded-lg shadow-md p-6 flex flex-col items-center justify-center space-y-4 mt-8 mx-auto max-w-lg">
        <p className="text-red-700 font-medium">{error}</p>
        <button onClick={() => navigate('/consignments')} className="text-blue-600 hover:underline">Return to Home</button>
      </div>
    )
  }

  return (
    <div className="bg-white rounded-lg shadow-md p-10 flex flex-col items-center justify-center space-y-4 max-w-lg mx-auto mt-8">
      <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      <p className="text-gray-700 font-medium text-lg">Verifying your payment...</p>
      <p className="text-sm text-gray-500 text-center">We are waiting for the payment gateway to confirm your transaction. Please do not close or refresh this page.</p>
    </div>
  )
}
