import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { JsonForm } from '../components/JsonForm'
import type { JsonSchema, UISchemaElement } from '../components/JsonForm'
import { getTaskDetails, sendTaskCommand } from '../services/task'
import type { TaskDetails } from '../services/mocks/taskData'
import type { TaskCommand } from '../services/task'

interface TaskPayload {
  version: number
  content: {
    schema: JsonSchema
    uischema: UISchemaElement
  }
}

export function TaskDetailScreen() {
  const { consignmentId, taskId } = useParams<{
    consignmentId: string
    taskId: string
  }>()
  const navigate = useNavigate()
  const [task, setTask] = useState<TaskDetails | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    async function fetchTask() {
      if (!consignmentId || !taskId) {
        setError('Consignment ID or Task ID is missing.')
        setLoading(false)
        return
      }

      try {
        setLoading(true)
        const taskDetails = await getTaskDetails(consignmentId, taskId)
        setTask(taskDetails)
      } catch (err) {
        setError('Failed to fetch task details.')
        console.error(err)
      } finally {
        setLoading(false)
      }
    }

    fetchTask()
  }, [consignmentId, taskId])

  const handleCommand = async (command: TaskCommand, formData: Record<string, unknown>) => {
    if (!task || !consignmentId || !taskId) return

    try {
      setSubmitting(true)
      setError(null)

      const response = await sendTaskCommand({
        command,
        taskId,
        consignmentId,
        data: formData,
      })

      if (response.success) {
        if (command === 'SUBMISSION') {
          alert('Task submitted successfully!')
          navigate(-1)
        } else {
          alert('Draft saved successfully!')
        }
      } else {
        setError(response.message || 'Operation failed.')
      }
    } catch (err) {
      console.error('Task command error:', err)
      setError('Operation failed. Please try again.')
    } finally {
      setSubmitting(false)
    }
  }

  const handleSubmit = async (formData: Record<string, unknown>) => {
    await handleCommand('SUBMISSION', formData)
  }

  const handleSaveDraft = async (formData: Record<string, unknown>) => {
    await handleCommand('DRAFT', formData)
  }

  if (loading) {
    return (
      <div className="flex justify-center items-center h-full">
        <p className="text-gray-500">Loading task...</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex justify-center items-center h-full">
        <p className="text-red-500">{error}</p>
      </div>
    )
  }

  if (!task) {
    return (
      <div className="flex justify-center items-center h-full">
        <p className="text-gray-500">Task not found.</p>
      </div>
    )
  }

  const payload = task.payload as TaskPayload
  const { schema, uischema } = payload.content

  return (
    <div className="p-4 sm:p-6 lg:p-8 bg-gray-50 min-h-full">
      <div className="max-w-4xl mx-auto">
        <div className="bg-white rounded-lg shadow-md p-6 mb-6">
          <h1 className="text-2xl font-bold text-gray-800">{task.name}</h1>
          <p className="text-gray-600 mt-2">{task.description}</p>
        </div>
        <div className="bg-white rounded-lg shadow-md p-6">
          <JsonForm
            schema={schema}
            uischema={uischema}
            onSubmit={handleSubmit}
            onSaveDraft={handleSaveDraft}
            submitLabel={submitting ? 'Submitting...' : 'Submit'}
            showDraftButton
          />
        </div>
      </div>
    </div>
  )
}