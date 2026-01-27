import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Button, Badge, Spinner, Text, Card, Flex, Box, TextField, TextArea, Callout } from '@radix-ui/themes'
import { ArrowLeftIcon, CheckCircledIcon, ExclamationTriangleIcon, InfoCircledIcon } from '@radix-ui/react-icons'
import { fetchApplicationDetail, approveTask, type OGAApplication, type ApproveRequest } from '../api'

export function ConsignmentDetailScreen() {
  const navigate = useNavigate()

  // Extract taskId from URL params
  const searchParams = new URLSearchParams(location.search)
  const taskId = searchParams.get('taskId')

  const [application, setApplication] = useState<OGAApplication | null>(null)
  const [loading, setLoading] = useState(true)
  const [formData, setFormData] = useState<Record<string, unknown>>({})
  const [reviewerName, setReviewerName] = useState('')
  const [decision, setDecision] = useState<'APPROVED' | 'REJECTED'>('APPROVED')
  const [comments, setComments] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)

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
    fetchData()
  }, [taskId])

  const handleFormChange = (field: string, value: unknown) => {
    setFormData((prev) => ({ ...prev, [field]: value }))
  }

  const handleApprove = async () => {
    if (!reviewerName.trim()) {
      setError('Reviewer name is required')
      return
    }

    if (!taskId || !application) {
      setError('Application data not available')
      return
    }

    setIsSubmitting(true)
    setError(null)

    try {
      const requestBody: ApproveRequest = {
        decision,
        comments: comments.trim() || undefined,
        reviewerName: reviewerName.trim(),
        formData: formData,
        consignmentId: application.consignmentId,
      }
      await approveTask(taskId, application.consignmentId, requestBody)
      setSuccess(true)
      setTimeout(() => navigate('/consignments'), 2000)
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
        <Button variant="soft" mt="4" onClick={() => navigate('/consignments')}>
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
        <Button variant="soft" mt="4" onClick={() => navigate('/consignments')}>
          <ArrowLeftIcon /> Back to List
        </Button>
      </Box>
    )
  }

  return (
    <div className="animate-fade-in max-w-5xl mx-auto">
      <Flex justify="between" align="center" mb="6">
        <Button variant="ghost" color="gray" onClick={() => navigate('/consignments')}>
          <ArrowLeftIcon /> Back to Consignments
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
                <Text size="1" color="gray" as="div" mb="1">Task ID</Text>
                <Text size="2" weight="medium" className="break-all font-mono">{application.taskId}</Text>
              </Box>
              <Box>
                <Text size="1" color="gray" as="div" mb="1">Consignment ID</Text>
                <Text size="2" weight="medium" className="break-all font-mono">{application.consignmentId}</Text>
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
              )}
            </div>
          </Card>

          {application.reviewerNotes && application.status !== 'PENDING' && (
            <Card size="2">
              <Text size="2" weight="bold" color="gray" mb="3" as="div" className="uppercase tracking-wider">
                Reviewer Notes
              </Text>
              <Text size="2" className="whitespace-pre-wrap">{application.reviewerNotes}</Text>
            </Card>
          )}
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
                          {typeof value === 'object' ? JSON.stringify(value) : String(value)}
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

              {application.status === 'PENDING' && (
                <>
                  <div className="border-t border-gray-100 my-4"></div>

                  <Box>
                    <Text as="label" size="2" weight="bold" mb="1" className="block">Reviewer Name *</Text>
                    <TextField.Root
                      placeholder="Enter your full name"
                      value={reviewerName}
                      onChange={(e) => setReviewerName(e.target.value)}
                      disabled={isSubmitting || success}
                      size="3"
                    />
                  </Box>

                  {/* Optional Review Fields */}
                  <div className="space-y-4 p-4 bg-blue-50/30 rounded-xl border border-blue-100/50">
                    <Text size="2" weight="bold" color="blue" mb="2" as="div" className="uppercase tracking-wider flex items-center gap-2">
                      <div className="w-1 h-4 bg-blue-500 rounded-full" />
                      Review Information (Optional)
                    </Text>

                    <Box>
                      <Text as="label" size="2" weight="bold" mb="1" className="block">Certificate Number</Text>
                      <TextField.Root
                        value={(formData.certificateNumber as string) || ''}
                        onChange={(e) => handleFormChange('certificateNumber', e.target.value)}
                        disabled={isSubmitting || success}
                        size="2"
                        placeholder="e.g., CERT-2024-001"
                      />
                    </Box>

                    <Box>
                      <Text as="label" size="2" weight="bold" mb="1" className="block">Inspection Date</Text>
                      <TextField.Root
                        type="date"
                        value={(formData.inspectionDate as string) || ''}
                        onChange={(e) => handleFormChange('inspectionDate', e.target.value)}
                        disabled={isSubmitting || success}
                        size="2"
                      />
                    </Box>

                    <Box>
                      <Flex align="center" gap="2">
                        <input
                          type="checkbox"
                          className="w-4 h-4 text-blue-600 rounded"
                          checked={(formData.verified as boolean) || false}
                          onChange={(e) => handleFormChange('verified', e.target.checked)}
                          disabled={isSubmitting || success}
                        />
                        <Text size="2" weight="bold">All documents verified</Text>
                      </Flex>
                    </Box>
                  </div>

                  <Box>
                    <Text as="label" size="2" weight="bold" mb="1" className="block">Final Decision *</Text>
                    <Flex gap="4" mt="2">
                      <Button
                        size="3"
                        variant={decision === 'APPROVED' ? 'solid' : 'soft'}
                        color="green"
                        className="flex-1 cursor-pointer"
                        onClick={() => setDecision('APPROVED')}
                        disabled={isSubmitting || success}
                      >
                        Approve
                      </Button>
                      <Button
                        size="3"
                        variant={decision === 'REJECTED' ? 'solid' : 'soft'}
                        color="red"
                        className="flex-1 cursor-pointer"
                        onClick={() => setDecision('REJECTED')}
                        disabled={isSubmitting || success}
                      >
                        Reject
                      </Button>
                    </Flex>
                  </Box>

                  <Box>
                    <Text as="label" size="2" weight="bold" mb="1" className="block">Comments</Text>
                    <TextArea
                      placeholder="Provide details about your decision..."
                      value={comments}
                      onChange={(e) => setComments(e.target.value)}
                      disabled={isSubmitting || success}
                      rows={4}
                      size="3"
                    />
                  </Box>

                  <div className="pt-4 border-t border-gray-100">
                    <Button
                      size="4"
                      className="w-full cursor-pointer"
                      onClick={handleApprove}
                      loading={isSubmitting}
                      disabled={success || !reviewerName.trim()}
                    >
                      {success ? 'Review Submitted' : 'Submit Final Review'}
                    </Button>
                  </div>
                </>
              )}
            </div>
          </Card>
        </div>
      </div>
    </div>
  )
}