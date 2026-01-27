// API service for OGA Portal

const API_BASE_URL = (import.meta.env.VITE_OGA_API_BASE_URL as string | undefined) ?? 'http://localhost:8081';

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
  serviceUrl: string;
  data: Record<string, unknown>;
  status: string;
  reviewerNotes?: string;
  reviewedAt?: string;
  createdAt: string;
  updatedAt: string;
}

// Fetch all consignments pending OGA review from OGA Service
export async function fetchPendingApplications(status?: string, signal?: AbortSignal): Promise<OGAApplication[]> {
  const url = status
    ? `${API_BASE_URL}/api/oga/applications?status=${status}`
    : `${API_BASE_URL}/api/oga/applications`;

  const response = await fetch(url, { signal });
  if (!response.ok) {
    throw new Error(`Failed to fetch pending applications: ${response.statusText}`);
  }

  return response.json() as Promise<OGAApplication[]>;
}

// Fetch application detail by taskId from OGA Service
export async function fetchApplicationDetail(taskId: string, signal?: AbortSignal): Promise<OGAApplication> {
  const response = await fetch(`${API_BASE_URL}/api/oga/applications/${taskId}`, { signal });
  if (!response.ok) {
    throw new Error(`Failed to fetch application: ${response.statusText}`);
  }
  return response.json() as Promise<OGAApplication>;
}

// Fetch consignment details including tasks and forms
export async function fetchConsignmentDetail(consignmentId: string, taskId?: string, signal?: AbortSignal): Promise<ConsignmentDetail> {
  try {
    const response = await fetch(`${API_BASE_URL}/api/consignments/${consignmentId}`, { signal });
    if (!response.ok) {
      throw new Error(`Failed to fetch consignment: ${response.statusText}`);
    }

    const consignment = await response.json() as Consignment;

    // Find all OGA_FORM tasks in the consignment that need review
    const ogaTasks: Task[] = [];
    consignment.items.forEach(item => {
      item.steps.forEach(step => {
        // OGA_FORM tasks are READY when waiting for OGA officer review
        if (step.type === 'OGA_FORM' && (step.status === 'READY' || step.status === 'IN_PROGRESS')) {
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
          const data = await traderFormResponse.json() as Record<string, unknown>;
          // Set traderForm if we got data with actual form fields (not just message/status)
          if (data && typeof data === 'object') {
            // Check if it's a placeholder message or actual form data
            if ('message' in data && 'status' in data && Object.keys(data).length === 2) {
              // This is a placeholder, don't set traderForm
              console.warn('Trader form not available:', data);
            } else {
              // This is actual form data or mock data with fields
              traderForm = data;
            }
          }
        } else {
          const errorText = await traderFormResponse.text();
          console.warn(`Failed to fetch trader form: ${traderFormResponse.status} ${errorText}`);
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

// Submit approval for a task via OGA Service
// OGA Service sends callback to the originating service
export async function approveTask(
  taskId: string,
  _consignmentId: string,
  requestBody: ApproveRequest,
  signal?: AbortSignal
): Promise<ApproveResponse> {
  // Build reviewer notes from comments and reviewer name
  const reviewerNotes = [
    `Reviewer: ${requestBody.reviewerName}`,
    requestBody.comments ? `Comments: ${requestBody.comments}` : '',
    requestBody.formData && Object.keys(requestBody.formData).length > 0
      ? `Form Data: ${JSON.stringify(requestBody.formData)}`
      : ''
  ].filter(Boolean).join('\n');

  const response = await fetch(`${API_BASE_URL}/api/oga/applications/${taskId}/review`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      decision: requestBody.decision,
      reviewerNotes: reviewerNotes,
    }),
    signal,
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({ error: response.statusText })) as { error?: string };
    throw new Error(errorData.error ?? `Failed to submit review: ${response.statusText}`);
  }

  return response.json() as Promise<ApproveResponse>;
}
