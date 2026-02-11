import { useState, useEffect, useMemo } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Button, Badge, Spinner, Text, Card, Flex, Box, Callout } from '@radix-ui/themes'
import { ArrowLeftIcon, CheckCircledIcon, ExclamationTriangleIcon, InfoCircledIcon } from '@radix-ui/react-icons'
import { fetchApplicationDetail, approveTask, type OGAApplication, type ApproveRequest, type Decision } from '../api'
import { JsonForm } from '@lsf/ui'
import type { JsonSchema, UISchemaElement } from '@jsonforms/core'

export function WorkflowDetailScreen() {
  const navigate = useNavigate()

  // Extract taskId from URL params
  const [searchParams] = useSearchParams()
  const taskId = searchParams.get('taskId')

  const [application, setApplication] = useState<OGAApplication | null>(null)
  const [loading, setLoading] = useState(true)
  const [formData, setFormData] = useState<Record<string, unknown>>({})
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)

  const parsedNotes = useMemo(() => {
    if (!application?.reviewerNotes) return { reviewerName: '', comments: '' };

    const lines = application.reviewerNotes.split('\n');
    const notes: Record<string, string> = {};

    lines.forEach(line => {
      const [key, ...rest] = line.split(': ');
      if (key && rest.length > 0) {
        notes[key.trim()] = rest.join(': ').trim();
      }
    });

    return {
      reviewerName: notes['Reviewer'] || '',
      comments: notes['Comments'] || ''
    };
  }, [application?.reviewerNotes]);

  useEffect(() => {
    async function fetchData() {
      if (!taskId) {
        setError('No task ID provided')
        setLoading(false)
        return
      }

      try {
        const data = await fetchApplicationDetail(taskId)
        setApplication(data)
      } catch (err) {
        setError('Failed to load application details')
        console.error(err)
      } finally {
        setLoading(false)
      }
    }
    void fetchData()
  }, [taskId])

  const handleFormChange = (data: Record<string, unknown>) => {
    setFormData(data)
  }

  const handleSubmit = async (data: Record<string, unknown>) => {
    if (!taskId || !application) {
      setError('Application data not available')
      return
    }

    const decision = data.decision as Decision
    const reviewerName = data.reviewerName as string
    const comments = data.comments as string | undefined

    // Explicit validation for required fields
    if (!decision) {
      setError('Please select a Final Decision');
      return;
    }
    if (!reviewerName || reviewerName.trim() === '') {
      setError('Please enter the Reviewer Name');
      return;
    }

    setIsSubmitting(true)
    setError(null)

    try {
      const requestBody: ApproveRequest = {
        decision: decision,
        comments: comments?.trim(),
        reviewerName: reviewerName.trim(),
        formData: data,
        workflowId: application.workflowId,
      }
      await approveTask(taskId, application.workflowId, requestBody)
      setSuccess(true)
      setTimeout(() => { void navigate('/workflows') }, 2000)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit review')
    } finally {
      setIsSubmitting(false)
    }
  }

  if (loading) {
    return (
      <Flex align="center" justify="center" py="9">
        <Spinner size="3" />
        <Text size="3" color="gray" ml="3">Loading application details...</Text>
      </Flex>
    )
  }

  if (error && !application) {
    return (
      <Box p="6">
        <Callout.Root color="red">
          <Callout.Icon><ExclamationTriangleIcon /></Callout.Icon>
          <Callout.Text>{error}</Callout.Text>
        </Callout.Root>
        <Button variant="soft" mt="4" onClick={() => { void navigate('/workflows') }}>
          <ArrowLeftIcon /> Back to List
        </Button>
      </Box>
    )
  }

  if (!application) {
    return (
      <Box p="6">
        <Callout.Root color="red">
          <Callout.Icon><ExclamationTriangleIcon /></Callout.Icon>
          <Callout.Text>Application not found</Callout.Text>
        </Callout.Root>
        <Button variant="soft" mt="4" onClick={() => { void navigate('/workflows') }}>
          <ArrowLeftIcon /> Back to List
        </Button>
      </Box>
    )
  }

  return (
    <div className="animate-fade-in max-w-5xl mx-auto px-4 py-6">
      <Flex justify="between" align="center" mb="6">
        <Button variant="ghost" color="gray" onClick={() => { void navigate('/workflows') }}>
          <ArrowLeftIcon /> Back to Workflows
        </Button>
        <Flex gap="3">
          <Badge size="2" color={
            application.status === 'APPROVED' ? 'green' :
              application.status === 'REJECTED' ? 'red' :
                'blue'
          } highContrast>
            {application.status}
          </Badge>
        </Flex>
      </Flex>

      {error && (
        <Callout.Root color="red" mb="6">
          <Callout.Icon><ExclamationTriangleIcon /></Callout.Icon>
          <Callout.Text>{error}</Callout.Text>
        </Callout.Root>
      )}

      {success && (
        <Callout.Root color="green" mb="6">
          <Callout.Icon><CheckCircledIcon /></Callout.Icon>
          <Callout.Text>Review submitted successfully! Redirecting...</Callout.Text>
        </Callout.Root>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left Column: Info */}
        <div className="lg:col-span-1 space-y-6">
          <Card size="2">
            <Text size="2" weight="bold" color="gray" mb="3" as="div" className="uppercase tracking-wider">
              Application Details
            </Text>
            <div className="space-y-4 mt-4">
              <Box>
                <Text size="1" color="gray" as="div" mb="1">Reference Number</Text>
                <Text size="2" weight="medium" className="break-all font-mono">{taskId}</Text>
              </Box>
              <Box>
                <Text size="1" color="gray" as="div" mb="1">Workflow ID</Text>
                <Text size="2" weight="medium" className="break-all font-mono">{application.workflowId}</Text>
              </Box>
              <Box>
                <Text size="1" color="gray" as="div" mb="1">Status</Text>
                <Badge size="2" color={
                  application.status === 'APPROVED' ? 'green' :
                    application.status === 'REJECTED' ? 'red' :
                      'blue'
                }>
                  {application.status}
                </Badge>
              </Box>
              <Box>
                <Text size="1" color="gray" as="div" mb="1">Submitted On</Text>
                <Text size="2" weight="medium">
                  {(() => {
                    const date = new Date(application.createdAt)
                    const datePart = date.toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' })
                    const timePart = date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: true })
                    return `${datePart} at ${timePart}`
                  })()}
                </Text>
              </Box>
              {application.reviewedAt && (
                <>
                  <Box>
                    <Text size="1" color="gray" as="div" mb="1">Reviewer Name</Text>
                    <Text size="2" weight="medium">{parsedNotes.reviewerName || '-'}</Text>
                  </Box>
                  <Box>
                    <Text size="1" color="gray" as="div" mb="1">Final Decision</Text>
                    <Text size="2" weight="medium">{application.status !== 'PENDING' ? application.status : '-'}</Text>
                  </Box>
                  <Box>
                    <Text size="1" color="gray" as="div" mb="1">Reviewer Comments</Text>
                    <Text size="2" weight="medium" className="whitespace-pre-wrap">{parsedNotes.comments || '-'}</Text>
                  </Box>
                  <Box>
                    <Text size="1" color="gray" as="div" mb="1">Reviewed On</Text>
                    <Text size="2" weight="medium">
                      {(() => {
                        const date = new Date(application.reviewedAt)
                        const datePart = date.toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' })
                        const timePart = date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: true })
                        return `${datePart} at ${timePart}`
                      })()}
                    </Text>
                  </Box>
                </>
              )}
            </div>
          </Card>
        </div>

        {/* Right Column: Review Form */}
        <div className="lg:col-span-2">
          <Card size="3">
            <Flex align="center" gap="2" mb="4">
              <InfoCircledIcon className="text-primary-600 w-5 h-5" />
              <Text size="4" weight="bold">
                {application.status === 'PENDING' ? 'Review Application' : 'Application Details'}
              </Text>
            </Flex>

            {application.status !== 'PENDING' ? (
              <Callout.Root color={application.status === 'APPROVED' ? 'green' : 'red'} mb="6">
                <Callout.Icon>
                  {application.status === 'APPROVED' ? <CheckCircledIcon /> : <ExclamationTriangleIcon />}
                </Callout.Icon>
                <Callout.Text>
                  This application has been {application.status.toLowerCase()}.
                </Callout.Text>
              </Callout.Root>
            ) : null}

            <div className="space-y-6 mt-6">
              {/* Submitted Data Section */}
              <div className="bg-gray-50 rounded-lg p-5 border border-gray-200">
                <Text size="2" weight="bold" color="gray" mb="4" as="div" className="uppercase tracking-wider flex items-center gap-2">
                  <InfoCircledIcon />
                  Submitted Information
                </Text>

                {application.data && Object.keys(application.data).length > 0 ? (
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {Object.entries(application.data).map(([key, value]) => (
                      <Box key={key} className="bg-white p-3 rounded border border-gray-100">
                        <Text size="1" color="gray" as="div" className="capitalize mb-1">
                          {key.replace(/([A-Z])/g, ' $1').replace(/_/g, ' ')}
                        </Text>
                        <Text size="2" weight="medium">
                          {typeof value === 'object' && value !== null ? JSON.stringify(value) : String(value)}
                        </Text>
                      </Box>
                    ))}
                  </div>
                ) : (
                  <Text size="2" color="gray" className="italic text-center py-2">
                    No submission data available
                  </Text>
                )}
              </div>

              {/* Review Section */}
              <div className="mt-6">
                {application.ogaForm ? (
                  <JsonForm
                    schema={application.ogaForm.schema as JsonSchema}
                    uiSchema={application.ogaForm.uiSchema as unknown as UISchemaElement}
                    data={formData}
                    onSubmit={(data) => { void handleSubmit(data) }}
                    onChange={handleFormChange}
                    readonly={application.status !== 'PENDING' || isSubmitting}
                    submitting={isSubmitting}
                    submitLabel={application.status === 'PENDING' ? "Submit Final Review" : "Update Review"}
                  />
                ) : (
                  <Callout.Root color="amber">
                    <Callout.Icon><InfoCircledIcon /></Callout.Icon>
                    <Callout.Text>No review form configuration available for this application.</Callout.Text>
                  </Callout.Root>
                )}
              </div>
            </div>
          </Card>
        </div>
      </div>
    </div>
  )
}