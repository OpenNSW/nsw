import { useState, useEffect } from 'react'
import {
    Button,
    Card,
    Heading,
    Text,
    Badge,
    Spinner,
    Flex,
    Box,
    Callout
} from '@radix-ui/themes'
import {
    FileTextIcon,
    PlayIcon,
    ArrowLeftIcon,
    EyeOpenIcon,
    CheckCircledIcon,
    ExclamationTriangleIcon
} from '@radix-ui/react-icons'
import { JsonForm } from '../components/JsonForm'
import {
    getTraderPreConsignments,
    createPreConsignment,
    getPreConsignment,
    fetchPreConsignmentTaskForm,
    submitPreConsignmentTask,
    type TraderPreConsignmentItem,
    type PreConsignmentInstance,
    type WorkflowNode
} from '../services/preConsignment'
import type { TaskFormData } from '../services/task'

export function PreconsignmentScreen() {
    const [loading, setLoading] = useState(true)
    const [items, setItems] = useState<TraderPreConsignmentItem[]>([])
    const [activeInstance, setActiveInstance] = useState<PreConsignmentInstance | null>(null)
    const [activeTaskId, setActiveTaskId] = useState<string | null>(null)
    const [formData, setFormData] = useState<TaskFormData | null>(null)
    const [formLoading, setFormLoading] = useState(false)
    const [isReadOnly, setIsReadOnly] = useState(false)
    const [notification, setNotification] = useState<{ type: 'success' | 'error', message: string } | null>(null)

    const loadData = async () => {
        try {
            setLoading(true)
            const response = await getTraderPreConsignments()
            setItems(response.items || [])
        } catch (error) {
            console.error('Failed to load pre-consignments', error)
            setNotification({ type: 'error', message: 'Failed to load pre-consignments list.' })
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        loadData()
    }, [])

    // Auto-dismiss success notifications
    useEffect(() => {
        if (notification?.type === 'success') {
            const timer = setTimeout(() => setNotification(null), 5000)
            return () => clearTimeout(timer)
        }
    }, [notification])

    const handleStartProcess = async (templateId: string) => {
        setNotification(null)
        try {
            setLoading(true)
            const instance = await createPreConsignment(templateId)
            await handleContinueProcess(instance)
            loadData()
        } catch (error) {
            console.error('Failed to start process', error)
            setNotification({ type: 'error', message: "Failed to start registration process." })
            setLoading(false)
        }
    }

    const handleContinueProcess = async (instanceStub: PreConsignmentInstance) => {
        setNotification(null)
        try {
            setLoading(true)

            const fullInstance = await getPreConsignment(instanceStub.id)
            const nodes = fullInstance.workflowNodes || []
            const isCompleted = fullInstance.state === 'COMPLETED';

            let targetNode: WorkflowNode | undefined = nodes.find(
                (node) => node.state === 'IN_PROGRESS' || node.state === 'READY'
            )

            if (!targetNode && isCompleted && nodes.length > 0) {
                targetNode = nodes[nodes.length - 1];
            }

            if (targetNode) {
                const formResponse = await fetchPreConsignmentTaskForm(fullInstance.id, targetNode.id)

                if (formResponse.success && formResponse.data) {
                    setFormData(formResponse.data)
                    setActiveTaskId(targetNode.id)
                    setActiveInstance(fullInstance)
                    setIsReadOnly(isCompleted)
                } else {
                    console.error("API returned success=false or no data")
                    setNotification({ type: 'error', message: "Failed to load task form data." })
                }
            } else {
                console.log('No suitable task found to view.')
                loadData()
            }
        } catch (error) {
            console.error('Failed to load process details', error)
            setNotification({ type: 'error', message: "An error occurred while loading the process details." })
        } finally {
            setLoading(false)
        }
    }

    const handleSubmit = async (data: unknown) => {
        if (!activeInstance || !activeTaskId) return
        setNotification(null)
        setFormLoading(true)

        try {
            const response = await submitPreConsignmentTask({
                command: 'SUBMISSION',
                preConsignmentId: activeInstance.id,
                taskId: activeTaskId,
                data: data as Record<string, unknown>,
            })

            if (response.success) {
                // Poll for completion to avoid race condition with async backend
                const maxRetries = 10;
                let attempts = 0;
                let isCompleted = false;

                while (attempts < maxRetries && !isCompleted) {
                    try {
                        const updatedInstance = await getPreConsignment(activeInstance.id);
                        if (updatedInstance.state === 'COMPLETED') {
                            isCompleted = true;
                        } else {
                            await new Promise(resolve => setTimeout(resolve, 500)); // Wait 500ms
                            attempts++;
                        }
                    } catch (e) {
                        console.warn("Polling failed, retrying...", e);
                        attempts++;
                    }
                }

                setActiveInstance(null)
                setActiveTaskId(null)
                setFormData(null)
                setNotification({ type: 'success', message: 'Registration submitted successfully.' })
                await loadData()
            } else {
                setNotification({ type: 'error', message: response.message || 'Submission failed' })
            }
        } catch (error) {
            console.error('Submission error', error)
            setNotification({ type: 'error', message: 'An unexpected error occurred during submission.' })
        } finally {
            setFormLoading(false)
        }
    }

    const handleBack = () => {
        setActiveInstance(null)
        setActiveTaskId(null)
        setFormData(null)
        setIsReadOnly(false)
        setNotification(null)
        loadData()
    }

    // Render logic for notifications
    const renderNotification = () => {
        if (!notification) return null;
        return (
            <Callout.Root color={notification.type === 'success' ? 'green' : 'red'} mb="4">
                <Callout.Icon>
                    {notification.type === 'success' ? <CheckCircledIcon /> : <ExclamationTriangleIcon />}
                </Callout.Icon>
                <Callout.Text>
                    {notification.message}
                </Callout.Text>
            </Callout.Root>
        )
    }

    if (loading && !activeInstance) {
        return (
            <Flex align="center" justify="center" style={{ height: '50vh' }}>
                <Spinner size="3" />
            </Flex>
        )
    }

    if (activeInstance && formData) {
        return (
            <Box p="6" className="max-w-4xl mx-auto bg-gray-50 min-h-full">
                <Button variant="ghost" mb="4" onClick={handleBack} style={{ cursor: 'pointer' }}>
                    <ArrowLeftIcon /> Back to List
                </Button>

                <Card size="3">
                    <Flex justify="between" align="center" mb="4">
                        <Heading size="6">{formData.title}</Heading>
                        {isReadOnly && <Badge color="green">Read Only</Badge>}
                    </Flex>

                    {renderNotification()}

                    {/* formLoading is incorrectly checked here, it replaces the form. 
                        We want the form to stay visible but show spinner on button.
                        Removing the spinner-replacement logic. */}
                    <JsonForm
                        schema={formData.schema}
                        uiSchema={formData.uiSchema}
                        data={formData.formData}
                        onSubmit={(data) => { handleSubmit(data).catch(err => console.error('Failed to handle submission:', err)); }}
                        submitLabel="Submit Registration"
                        showAutoFillButton={import.meta.env.VITE_SHOW_AUTOFILL_BUTTON === 'true' && !isReadOnly}
                        readonly={isReadOnly}
                        submitting={formLoading}
                    />
                </Card>
            </Box>
        )
    }

    return (
        <Box p="6">
            <Heading mb="6">Pre-Consignment Registration</Heading>

            {renderNotification()}

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {items.map((item) => {
                    const hasInstance = !!item.preConsignment
                    const isCompleted = item.state === 'COMPLETED'
                    const isLocked = item.state === 'LOCKED'
                    const isInProgress = item.state === 'IN_PROGRESS' || (hasInstance && !isCompleted)

                    return (
                        <Card key={item.id} size="2" style={{ position: 'relative' }}>
                            <Flex direction="column" gap="3">
                                <Flex justify="between" align="start">
                                    <Box>
                                        <Heading size="4" mb="1">{item.name}</Heading>
                                        <Text size="2" color="gray">{item.description}</Text>
                                    </Box>
                                    <FileTextIcon width="24" height="24" className="text-gray-400" />
                                </Flex>

                                <Flex justify="between" align="center" mt="4">
                                    <Badge
                                        color={
                                            isCompleted ? 'green' :
                                                isInProgress ? 'blue' :
                                                    isLocked ? 'gray' : 'orange'
                                        }
                                    >
                                        {item.state.replace('_', ' ')}
                                    </Badge>

                                    {!hasInstance ? (
                                        <Button
                                            onClick={() => handleStartProcess(item.id)}
                                            disabled={isLocked}
                                            style={{ cursor: isLocked ? 'not-allowed' : 'pointer' }}
                                        >
                                            <PlayIcon /> Start
                                        </Button>
                                    ) : isCompleted ? (
                                        <Button
                                            variant="outline"
                                            color="green"
                                            onClick={() => handleContinueProcess(item.preConsignment!)}
                                            style={{ cursor: 'pointer' }}
                                        >
                                            <EyeOpenIcon /> View
                                        </Button>
                                    ) : (
                                        <Button onClick={() => handleContinueProcess(item.preConsignment!)} style={{ cursor: 'pointer' }}>
                                            Continue
                                        </Button>
                                    )}
                                </Flex>
                            </Flex>
                        </Card>
                    )
                })}
            </div>

            {items.length === 0 && (
                <Text color="gray" align="center" as="p" mt="9">
                    No registration templates available at this time.
                </Text>
            )}
        </Box>
    )
}