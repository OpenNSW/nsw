// API service for OGA Portal

const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? 'http://localhost:8080';
const OGA_API_BASE_URL = (import.meta.env.VITE_OGA_API_BASE_URL as string | undefined) ?? 'http://localhost:8081';

export interface Consignment {
  id: string;
  traderId: string;
  tradeFlow: 'IMPORT' | 'EXPORT';
  state: string;
  items: Array<{
    hsCodeID: string;
    steps: Array<{
      stepId: string;
      type: string;
      taskId: string;
      status: string;
      dependsOn: string[];
    }>;
  }>;
  createdAt: string;
  updatedAt: string;
}

export interface Task {
  id: string;
  consignmentId: string;
  stepId: string;
  type: string;
  status: string;
  config: Record<string, unknown>;
  dependsOn: Record<string, string>;
}

export interface FormResponse {
  id: string;
  name: string;
  schema: Record<string, unknown>;
  uiSchema: Record<string, unknown>;
  version: string;
}

export interface ConsignmentDetail extends Consignment {
  ogaTasks: Task[];
  traderForm?: Record<string, unknown>;
  ogaForm?: FormResponse;
}

export type Decision = 'APPROVED' | 'REJECTED';

export interface ApproveRequest {
  formData: Record<string, unknown>;
  consignmentId: string;
  decision: Decision;
  reviewerName: string;
  comments?: string;
}

export interface ApproveResponse {
  success: boolean;
  message?: string;
  error?: string;
}

export interface OGAApplication {
  taskId: string;
  consignmentId: string;
  formId: string;
  status: string;
}

// Fetch all applications pending OGA review from OGA service
export async function fetchPendingApplications(signal?: AbortSignal): Promise<OGAApplication[]> {
  try {
    const response = await fetch(`${OGA_API_BASE_URL}/api/oga/applications`, { signal });
    if (!response.ok) {
      throw new Error(`Failed to fetch pending applications: ${response.statusText}`);
    }

    return await response.json() as OGAApplication[];
  } catch (error) {
    if (signal?.aborted) throw error;
    console.warn('Failed to fetch from OGA backend, returning MOCK data:', error);

    return [
      {
        taskId: '550e8400-e29b-41d4-a716-446655440003',
        consignmentId: '550e8400-e29b-41d4-a716-446655440000',
        formId: 'oga-export-permit',
        status: 'IN_PROGRESS',
      }
    ];
  }
}

// Fetch consignment details including tasks and forms
export async function fetchConsignmentDetail(consignmentId: string, taskId?: string, signal?: AbortSignal): Promise<ConsignmentDetail> {
  try {
    const response = await fetch(`${API_BASE_URL}/api/consignments/${consignmentId}`, { signal });
    if (!response.ok) {
      throw new Error(`Failed to fetch consignment: ${response.statusText}`);
    }

    const consignment = await response.json() as Consignment;

    // Find all OGA_FORM tasks in the consignment
    const ogaTasks: Task[] = [];
    consignment.items.forEach(item => {
      item.steps.forEach(step => {
        if (step.type === 'OGA_FORM' && step.status === 'IN_PROGRESS') {
          ogaTasks.push({
            id: step.taskId,
            consignmentId: consignment.id,
            stepId: step.stepId,
            type: step.type,
            status: step.status,
            config: {},
            dependsOn: {},
          });
        }
      });
    });

    // Determine which task to fetch forms for
    const targetTaskId = taskId || (ogaTasks.length > 0 ? ogaTasks[0].id : undefined);

    // Get trader form submission and OGA form
    let traderForm: Record<string, unknown> | undefined;
    let ogaForm: FormResponse | undefined;

    if (targetTaskId) {
      // Fetch trader form submission
      try {
        const traderFormResponse = await fetch(`${API_BASE_URL}/api/tasks/${targetTaskId}/trader-form`, { signal });
        if (traderFormResponse.ok) {
          traderForm = await traderFormResponse.json() as Record<string, unknown>;
        }
      } catch (error) {
        console.warn('Failed to fetch trader form:', error);
      }

      // Fetch OGA form schema
      try {
        const ogaFormResponse = await fetch(`${API_BASE_URL}/api/tasks/${targetTaskId}/form`, { signal });
        if (ogaFormResponse.ok) {
          ogaForm = await ogaFormResponse.json() as FormResponse;
        }
      } catch (error) {
        console.warn('Failed to fetch OGA form:', error);
      }
    }

    return {
      ...consignment,
      ogaTasks,
      traderForm,
      ogaForm,
    };
  } catch (error) {
    if (signal?.aborted) throw error;
    console.warn('Failed to fetch details, returning MOCK detail:', error);

    // Mock detail fallback for development
    if (consignmentId === '550e8400-e29b-41d4-a716-446655440000') {
      return {
        id: '550e8400-e29b-41d4-a716-446655440000',
        traderId: 'trader-123',
        tradeFlow: 'EXPORT',
        state: 'IN_PROGRESS',
        createdAt: '2024-01-17T10:00:00Z',
        updatedAt: new Date().toISOString(),
        items: [{
          hsCodeID: '0902.20.19',
          steps: [{
            stepId: 'oga-review',
            type: 'OGA_FORM',
            taskId: '550e8400-e29b-41d4-a716-446655440003',
            status: 'IN_PROGRESS',
            dependsOn: []
          }]
        }],
        ogaTasks: [{
          id: '550e8400-e29b-41d4-a716-446655440003',
          consignmentId: '550e8400-e29b-41d4-a716-446655440000',
          stepId: 'oga-review',
          type: 'OGA_FORM',
          status: 'IN_PROGRESS',
          config: {},
          dependsOn: {}
        }],
        traderForm: {
          exporterName: 'Sri Lanka Tea Exporters Ltd',
          destinationCountry: 'United Kingdom',
          netWeight: 5000,
          grossWeight: 5200,
          invoiceValue: 12500
        },
        ogaForm: {
          id: 'oga-export-permit',
          name: 'Tea Export Permit Review',
          version: '1.0',
          schema: {
            type: 'object',
            properties: {
              qualityCheck: { type: 'boolean', title: 'Quality Standards Met' },
              batchNumber: { type: 'string', title: 'Certified Batch Number' },
              remarks: { type: 'string', title: 'Officer Remarks' }
            }
          },
          uiSchema: {}
        }
      };
    }

    throw error;
  }
}

// Submit approval for a task
export async function approveTask(
  taskId: string,
  consignmentId: string,
  requestBody: ApproveRequest,
  signal?: AbortSignal
): Promise<ApproveResponse> {
  // Call the centralized task execution endpoint
  const response = await fetch(`${API_BASE_URL}/api/tasks`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      task_id: taskId,
      consignment_id: consignmentId,
      payload: {
        action: 'OGA_VERIFICATION',
        content: {
          reviewerName: requestBody.reviewerName,
          comments: requestBody.comments,
          decision: requestBody.decision,
          ...requestBody.formData,
        },
      },
    }),
    signal,
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({ error: response.statusText })) as { error?: string };
    throw new Error(errorData.error ?? `Failed to approve task: ${response.statusText}`);
  }

  return response.json() as Promise<ApproveResponse>;
}
