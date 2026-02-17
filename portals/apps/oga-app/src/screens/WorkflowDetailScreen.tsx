import {useState, useEffect} from 'react'
import {useNavigate, useSearchParams} from 'react-router-dom'
import {Button, Badge, Spinner, Text, Card, Flex, Box, Callout} from '@radix-ui/themes'
import {ArrowLeftIcon, CheckCircledIcon, ExclamationTriangleIcon, InfoCircledIcon} from '@radix-ui/react-icons'
import {fetchApplicationDetail, submitReview, type OGAApplication} from '../api'
import {JsonForm, useJsonForm, type UISchemaElement, type JsonSchema} from "../components/JsonForm";

const EMPTY_SCHEMA: JsonSchema = {type: 'object', properties: {}}

export function WorkflowDetailScreen() {
  const navigate = useNavigate()

  // Extract taskId from URL params
  const [searchParams] = useSearchParams()
  const taskId = searchParams.get('taskId')

  const [application, setApplication] = useState<OGAApplication | null>(null)
  const [loading, setLoading] = useState(true)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)

  const [formConfig, setFormConfig] = useState<{ schema: JsonSchema; uiSchema: UISchemaElement } | null>(null)

  const form = useJsonForm({
    schema: formConfig?.schema ?? EMPTY_SCHEMA,
    onSubmit: async (formValues) => {
      if (!taskId || !application) {
        setError('Application data not available')
        return
      }

      setIsSubmitting(true)
      setError(null)

      try {
        await submitReview(taskId, formValues)
        setSuccess(true)
        setTimeout(() => navigate('/workflows'), 2000)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to submit review')
      } finally {
        setIsSubmitting(false)
      }
    },
  })

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
        setFormConfig({schema: data.form.schema, uiSchema: data.form.uiSchema})
      } catch (err) {
        setError('Failed to load application details')
        console.error(err)
      } finally {
        setLoading(false)
      }
    }

    void fetchData()
  }, [taskId])


  if (loading) {
    return (
      <Flex align="center" justify="center" py="9">
        <Spinner size="3"/>
        <Text size="3" color="gray" ml="3">Loading application details...</Text>
      </Flex>
    )
  }

  if (error && !application) {
    return (
      <Box p="6">
        <Callout.Root color="red">
          <Callout.Icon><ExclamationTriangleIcon/></Callout.Icon>
          <Callout.Text>{error}</Callout.Text>
        </Callout.Root>
        <Button variant="soft" mt="4" onClick={() => {
          void navigate('/workflows')
        }}>
          <ArrowLeftIcon/> Back to List
        </Button>
      </Box>
    )
  }

  if (!application) {
    return (
      <Box p="6">
        <Callout.Root color="red">
          <Callout.Icon><ExclamationTriangleIcon/></Callout.Icon>
          <Callout.Text>Application not found</Callout.Text>
        </Callout.Root>
        <Button variant="soft" mt="4" onClick={() => {
          void navigate('/workflows')
        }}>
          <ArrowLeftIcon/> Back to List
        </Button>
      </Box>
    )
  }

  return (
    <div className="animate-fade-in max-w-5xl mx-auto">
      <Flex justify="between" align="center" mb="6">
        <Button variant="ghost" color="gray" onClick={() => {
          void navigate('/workflows')
        }}>
          <ArrowLeftIcon/> Back to Workflows
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
          <Callout.Icon><ExclamationTriangleIcon/></Callout.Icon>
          <Callout.Text>{error}</Callout.Text>
        </Callout.Root>
      )}

      {success && (
        <Callout.Root color="green" mb="6">
          <Callout.Icon><CheckCircledIcon/></Callout.Icon>
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
                    const datePart = date.toLocaleDateString('en-US', {month: 'long', day: 'numeric', year: 'numeric'})
                    const timePart = date.toLocaleTimeString('en-US', {
                      hour: '2-digit',
                      minute: '2-digit',
                      hour12: true
                    })
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
                      const datePart = date.toLocaleDateString('en-US', {
                        month: 'long',
                        day: 'numeric',
                        year: 'numeric'
                      })
                      const timePart = date.toLocaleTimeString('en-US', {
                        hour: '2-digit',
                        minute: '2-digit',
                        hour12: true
                      })
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
              <InfoCircledIcon className="text-primary-600 w-5 h-5"/>
              <Text size="4" weight="bold">
                {application.status === 'PENDING' ? 'Review Application' : 'Application Details'}
              </Text>
            </Flex>

            {application.status !== 'PENDING' ? (
              <Callout.Root color={application.status === 'APPROVED' ? 'green' : 'red'} mb="6">
                <Callout.Icon>
                  {application.status === 'APPROVED' ? <CheckCircledIcon/> : <ExclamationTriangleIcon/>}
                </Callout.Icon>
                <Callout.Text>
                  This application has been {application.status.toLowerCase()}.
                </Callout.Text>
              </Callout.Root>
            ) : null}

            <div className="space-y-6 mt-6">
              {/* Submitted Data Section */}
              <div className="bg-gray-50 rounded-lg p-5 border border-gray-200">
                <Text size="2" weight="bold" color="gray" mb="4" as="div"
                      className="uppercase tracking-wider flex items-center gap-2">
                  <InfoCircledIcon/>
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

              <div className="border-t border-gray-100 my-4"></div>

              {formConfig && <JsonForm
                  schema={formConfig?.schema}
                  uiSchema={formConfig?.uiSchema}
                  values={form.values}
                  touched={form.touched}
                  setValue={form.setValue}
                  setTouched={form.setTouched}
                  errors={form.errors}
              />
              }

              <Flex justify="end" gap="3" mt="6">
                <Button
                  variant="soft"
                  color="gray"
                  onClick={() => {
                    void navigate('/workflows')
                  }}
                  disabled={form.isSubmitting || isSubmitting}
                >
                  Cancel
                </Button>
                <Button
                  onClick={(e) => {
                    void form.handleSubmit(e as unknown as React.FormEvent)
                  }}
                  disabled={form.isSubmitting || isSubmitting}
                >
                  {form.isSubmitting || isSubmitting ? <Spinner size="1"/> : null}
                  Submit Review
                </Button>
              </Flex>
            </div>
          </Card>
        </div>
      </div>
    </div>
  )
}