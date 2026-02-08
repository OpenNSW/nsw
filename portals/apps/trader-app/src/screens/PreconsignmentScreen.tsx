import { useState } from 'react'
import { Card, Text, Flex, Button, Table, Badge, Dialog, Spinner, Callout } from '@radix-ui/themes'
import { CheckCircledIcon, ExclamationTriangleIcon, LockClosedIcon } from '@radix-ui/react-icons'
import { JsonForm } from '../components/JsonForm'
import type { JsonSchema, UISchemaElement } from '../components/JsonForm'

interface PreconsignmentProcess {
    id: number
    name: string
    actionBy: string
    dependencies: string
    respondingActor: string
    inputOutput: string
    status: 'READY' | 'LOCKED'
    formName: string // Used to fetch the JSON form template
}

const processes: PreconsignmentProcess[] = [
    {
        id: 1,
        name: 'Business Registration',
        actionBy: 'Exporter',
        dependencies: '',
        respondingActor: 'RoC / Divisional Secretariat',
        inputOutput: 'Input: BR Application\nOutput: BR No',
        status: 'READY',
        formName: 'Business Registration'
    },
    {
        id: 2,
        name: 'TIN Registration',
        actionBy: 'Exporter',
        dependencies: 'Business Registration',
        respondingActor: 'Inland Revenue Dept',
        inputOutput: 'Input: BR No\nOutput: TIN No',
        status: 'LOCKED',
        formName: 'TIN Registration'
    },
    {
        id: 3,
        name: 'VAT Registration',
        actionBy: 'Exporter',
        dependencies: 'Business Registration',
        respondingActor: 'Inland Revenue Dept',
        inputOutput: 'Input: BR No\nOutput: VAT No',
        status: 'LOCKED',
        formName: 'VAT Registration'
    },
    {
        id: 4,
        name: 'CDA Manufacturer Registration (1st Time)',
        actionBy: 'Exporter/Manufacturer',
        dependencies: 'Business Registration, TIN Registration, VAT Registration',
        respondingActor: 'CDA',
        inputOutput: 'Input: BR, TIN, VAT, EPL Certificate\nOutput: CDA Manufacturer Reg Certificate',
        status: 'LOCKED',
        formName: 'CDA Manufacturer Registration'
    },
    {
        id: 5,
        name: 'CDA Exporter Registration (1st time)',
        actionBy: 'Exporter',
        dependencies: 'Business Registration, TIN Registration, VAT Registration, CDA Manufacturer Registration',
        respondingActor: 'CDA',
        inputOutput: 'Input: Various Docs\nOutput: CDA Exporter Reg Certificate',
        status: 'LOCKED',
        formName: 'CDA Exporter Registration'
    },
    {
        id: 6,
        name: 'ASYCUDA World Registration (TIN/VAT Registration)',
        actionBy: 'Exporter',
        dependencies: 'Business Registration, TIN Registration, VAT Registration',
        respondingActor: 'Sri Lanka Customs',
        inputOutput: 'Input: BR, TIN, VAT, Docs\nOutput: ASYCUDA Reg / Custom Profile',
        status: 'LOCKED',
        formName: 'ASYCUDA World Registration'
    },
]

export function PreconsignmentScreen() {
    const [selectedProcess, setSelectedProcess] = useState<PreconsignmentProcess | null>(null)
    const [isDialogOpen, setIsDialogOpen] = useState(false)
    const [completedProcesses, setCompletedProcesses] = useState<Set<number>>(new Set())

    // Form fetching state
    const [formSchema, setFormSchema] = useState<JsonSchema | null>(null)
    const [uiSchema, setUiSchema] = useState<UISchemaElement | null>(null)
    const [formData, setFormData] = useState<Record<string, unknown>>({})
    const [isLoading, setIsLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)

    const getStatus = (process: PreconsignmentProcess) => {
        if (completedProcesses.has(process.id)) return 'COMPLETED'
        if (!process.dependencies) return 'READY'

        const deps = process.dependencies.split(',').map(d => d.trim()).filter(Boolean)
        const allDepsMet = deps.every(depName => {
            const depProcess = processes.find(p => p.name === depName)
            return depProcess && completedProcesses.has(depProcess.id)
        })

        return allDepsMet ? 'READY' : 'LOCKED'
    }

    const fetchFormTemplate = async (formName: string) => {
        setIsLoading(true)
        setError(null)
        setFormSchema(null)
        setUiSchema(null)

        try {
            // Using the new API endpoint
            const response = await fetch(`${import.meta.env.VITE_API_URL || 'http://localhost:8080'}/api/v1/forms?name=${encodeURIComponent(formName)}`)
            if (!response.ok) {
                throw new Error(`Failed to fetch form: ${response.statusText}`)
            }
            const data = (await response.json()) as { schema: JsonSchema; uiSchema: UISchemaElement }
            setFormSchema(data.schema)
            setUiSchema(data.uiSchema)
            setFormData({}) // Reset data
        } catch (err: unknown) {
            console.error(err)
            const errorMessage = err instanceof Error ? err.message : 'Error loading form.'
            setError(errorMessage)
        } finally {
            setIsLoading(false)
        }
    }

    const handleProcessClick = async (process: PreconsignmentProcess) => {
        setSelectedProcess(process)
        setIsDialogOpen(true)
        await fetchFormTemplate(process.formName)
    }

    const handleClose = () => {
        setIsDialogOpen(false)
        setSelectedProcess(null)
        setFormSchema(null)
    }

    const handleSubmit = (data: Record<string, unknown>) => {
        if (selectedProcess) {
            console.log('Submitted data for', selectedProcess.name, data)
            // Here you would typically save the data to backend

            setCompletedProcesses(prev => new Set(prev).add(selectedProcess.id))
            handleClose()
        }
    }

    return (
        <div className="p-6 max-w-7xl mx-auto animate-fade-in">
            <div className="mb-6">
                <Text size="6" weight="bold" as="p" className="text-gray-900 mb-2">Pre-Consignment Processes</Text>
                <Text size="2" color="gray" as="p">
                    Complete the following registration steps to proceed with export consignments.
                </Text>
            </div>

            <Card size="3">
                <Table.Root>
                    <Table.Header>
                        <Table.Row className="bg-gray-50">
                            <Table.ColumnHeaderCell>#</Table.ColumnHeaderCell>
                            <Table.ColumnHeaderCell>Process Name</Table.ColumnHeaderCell>
                            <Table.ColumnHeaderCell>Action Taken By</Table.ColumnHeaderCell>
                            <Table.ColumnHeaderCell>Dependencies</Table.ColumnHeaderCell>
                            <Table.ColumnHeaderCell>Responding Actor</Table.ColumnHeaderCell>
                            <Table.ColumnHeaderCell>Input/Output</Table.ColumnHeaderCell>
                            <Table.ColumnHeaderCell>Status</Table.ColumnHeaderCell>
                        </Table.Row>
                    </Table.Header>

                    <Table.Body>
                        {processes.map((process) => {
                            const status = getStatus(process)
                            const isLocked = status === 'LOCKED'

                            return (
                                <Table.Row
                                    key={process.id}
                                    className={isLocked ? "bg-gray-50 opacity-60 cursor-not-allowed" : "hover:bg-blue-50/50 cursor-pointer transition-colors"}
                                    onClick={() => { if (!isLocked) void handleProcessClick(process); }}
                                >
                                    <Table.Cell>{process.id}</Table.Cell>
                                    <Table.Cell>
                                        <Text weight="medium" color={isLocked ? 'gray' : undefined}>{process.name}</Text>
                                    </Table.Cell>
                                    <Table.Cell>{process.actionBy}</Table.Cell>
                                    <Table.Cell>
                                        <Text size="1" color="gray">{process.dependencies || '-'}</Text>
                                    </Table.Cell>
                                    <Table.Cell>{process.respondingActor}</Table.Cell>
                                    <Table.Cell>
                                        <Text size="1" className="whitespace-pre-line">{process.inputOutput}</Text>
                                    </Table.Cell>
                                    <Table.Cell>
                                        {status === 'COMPLETED' && <Badge color="green"><CheckCircledIcon /> Completed</Badge>}
                                        {status === 'READY' && <Badge color="blue">Ready</Badge>}
                                        {status === 'LOCKED' && <Badge color="gray"><LockClosedIcon /> Locked</Badge>}
                                    </Table.Cell>
                                </Table.Row>
                            )
                        })}
                    </Table.Body>
                </Table.Root>
            </Card>

            <Dialog.Root open={isDialogOpen} onOpenChange={setIsDialogOpen}>
                <Dialog.Content style={{ maxWidth: 600 }}>
                    <Dialog.Title>{selectedProcess?.name}</Dialog.Title>
                    <Dialog.Description size="2" mb="4">
                        Complete the form below.
                    </Dialog.Description>

                    {isLoading && (
                        <Flex align="center" justify="center" p="4">
                            <Spinner />
                            <Text ml="2">Loading form...</Text>
                        </Flex>
                    )}

                    {error && (
                        <Callout.Root color="red">
                            <Callout.Icon><ExclamationTriangleIcon /></Callout.Icon>
                            <Callout.Text>{error}</Callout.Text>
                        </Callout.Root>
                    )}

                    {!isLoading && !error && formSchema && (
                        <JsonForm
                            schema={formSchema}
                            uiSchema={uiSchema || undefined}
                            data={formData}
                            onSubmit={(data) => { void handleSubmit(data as Record<string, unknown>) }}
                            submitLabel="Submit & Complete"
                        />
                    )}

                    <Flex justify="end" mt="4">
                        <Dialog.Close>
                            <Button variant="soft" color="gray" onClick={() => { handleClose(); }}>Close</Button>
                        </Dialog.Close>
                    </Flex>
                </Dialog.Content>
            </Dialog.Root>
        </div>
    )
}
