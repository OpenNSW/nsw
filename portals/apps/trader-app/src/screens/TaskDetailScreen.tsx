import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { JsonForm } from '../components/JsonForm'
import { getTaskDetails } from '../services/task'
import type { TaskDetails } from '../services/mocks/taskData'

export function TaskDetailScreen() {
  const { consignmentId, taskId } = useParams<{
    consignmentId: string
    taskId: string
  }>()
  const [task, setTask] = useState<TaskDetails | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

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

  const handleSubmit = (formData: unknown) => {
    console.log('Form submitted!', formData)
    // Here you would typically call an API to save the data
    alert('Form submitted successfully! Check the console for the data.')
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

  return (
    <div className="p-4 sm:p-6 lg:p-8 bg-gray-50 min-h-full">
      <div className="max-w-4xl mx-auto">
        <div className="bg-white rounded-lg shadow-md p-6 mb-6">
          <h1 className="text-2xl font-bold text-gray-800">{task.name}</h1>
          <p className="text-gray-600 mt-2">{task.description}</p>
        </div>
        <div className="bg-white rounded-lg shadow-md p-6">
          <JsonForm
            schema={task.schema}
            uischema={task.uischema}
            onSubmit={handleSubmit}
            submitLabel="Complete Task"
          />
        </div>
      </div>
    </div>
  )
}
