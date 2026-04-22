/**
 * OGA-app–specific upload implementation. Points to this app's backend;
 * when OGA moves to a separate repo, this file can target OGA-specific endpoints/S3 without touching shared UI.
 */
import type { ApiClient } from '../api'

const API_BASE_URL = (import.meta.env.VITE_OGA_API_BASE_URL as string | undefined) ?? 'http://localhost:8080/api/v1'

export interface UploadResponse {
  key: string
  name: string
}

export async function uploadFile(apiClient: ApiClient, file: File): Promise<UploadResponse> {
  // Request presigned URL and metadata from backend
  const metadataResponse = await fetch(`${API_BASE_URL}/uploads`, {
    method: 'POST',
    headers: {
      ...(await apiClient.getAuthHeaders(false)),
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      filename: file.name,
      mime_type: file.type || 'application/octet-stream',
      size: file.size,
    }),
  })

  if (!metadataResponse.ok) {
    const errorText = await metadataResponse.text()
    console.error(`Metadata request error ${metadataResponse.status}: ${errorText}`)
    throw new Error(`Failed to initialize upload: ${metadataResponse.status} ${metadataResponse.statusText}`)
  }

  const meta = (await metadataResponse.json()) as { key: string; name: string; upload_url: string }

  // Upload file bytes directly to the storage destination (presigned URL)
  const uploadResponse = await fetch(meta.upload_url, {
    method: 'PUT',
    headers: {
      'Content-Type': file.type || 'application/octet-stream',
    },
    body: file,
  })

  if (!uploadResponse.ok) {
    const errorText = await uploadResponse.text()
    console.error(`Direct storage upload error ${uploadResponse.status}: ${errorText}`)
    throw new Error(`Failed to upload file to storage: ${uploadResponse.status} ${uploadResponse.statusText}`)
  }

  return { key: meta.key, name: meta.name }
}

export async function getDownloadUrl(apiClient: ApiClient, key: string): Promise<{ url: string; expiresAt: number }> {
  const response = await apiClient.get<{ download_url: string; expires_at: number }>(
    `/api/oga/uploads/${key}`
  )
  return { url: response.download_url, expiresAt: response.expires_at }
}
