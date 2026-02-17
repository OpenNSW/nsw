// API service for OGA Portal
import type {JsonSchema, UISchemaElement} from "./components/JsonForm";

const API_BASE_URL = (import.meta.env.VITE_OGA_API_BASE_URL as string | undefined) ?? 'http://localhost:8081';

export interface ReviewResponse {
  success: boolean;
  message?: string;
  error?: string;
}

export interface OGAApplication {
  taskId: string;
  workflowId: string;
  serviceUrl: string;
  data: Record<string, unknown>;
  meta?: {
    type: string;
    verificationId: string;
  };
  form: {
    schema: JsonSchema;
    uiSchema: UISchemaElement;
  };
  status: string;
  reviewerNotes?: string;
  reviewedAt?: string;
  createdAt: string;
  updatedAt: string;
  ogaForm?: {
    schema: Record<string, unknown>;
    uiSchema: Record<string, unknown>;
  };
}


export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}

export async function fetchApplications(
  params?: { status?: string; page?: number; pageSize?: number },
  signal?: AbortSignal
): Promise<PaginatedResponse<OGAApplication>> {
  const searchParams = new URLSearchParams();
  if (params?.status) searchParams.set('status', params.status);
  if (params?.page) searchParams.set('page', String(params.page));
  if (params?.pageSize) searchParams.set('pageSize', String(params.pageSize));

  const query = searchParams.toString();
  const url = `${API_BASE_URL}/api/oga/applications${query ? `?${query}` : ''}`;

  const response = await fetch(url, { signal });
  if (!response.ok) {
    throw new Error(`Failed to fetch pending applications: ${response.statusText}`);
  }

  return response.json() as Promise<PaginatedResponse<OGAApplication>>;
}

// Fetch application detail by taskId from OGA Service
export async function fetchApplicationDetail(taskId: string, signal?: AbortSignal): Promise<OGAApplication> {
  const response = await fetch(`${API_BASE_URL}/api/oga/applications/${taskId}`, { signal });
  if (!response.ok) {
    throw new Error(`Failed to fetch application: ${response.statusText}`);
  }
  return response.json() as Promise<OGAApplication>;
}

// Submit review for a task via OGA Service
export async function submitReview(
  taskId: string,
  formValues: Record<string, unknown>,
  signal?: AbortSignal
): Promise<ReviewResponse> {
  const response = await fetch(`${API_BASE_URL}/api/oga/applications/${taskId}/review`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(formValues),
    signal,
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({ error: response.statusText })) as { error?: string };
    throw new Error(errorData.error ?? `Failed to submit review: ${response.statusText}`);
  }

  return response.json() as Promise<ReviewResponse>;
}
