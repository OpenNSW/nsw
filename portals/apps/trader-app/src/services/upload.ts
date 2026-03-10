/**
 * Trader-app–specific upload implementation. Points to this app's backend;
 * when the API or auth changes, only this file is updated.
 */
import type { ApiClient } from './api'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1'

export interface UploadResponse {
  key: string
  name: string
}

export async function uploadFile(apiClient: ApiClient, file: File): Promise<UploadResponse> {
  const formData = new FormData()
  formData.append('file', file)

  const response = await fetch(`${API_BASE_URL}/uploads`, {
    method: 'POST',
    headers: await apiClient.getAuthHeaders(false),
    body: formData,
  })

  if (!response.ok) {
    const errorText = await response.text()
    console.error(`Upload error ${response.status}: ${errorText}`)
    throw new Error(`Failed to upload file: ${response.status} ${response.statusText}`)
  }

  const meta = (await response.json()) as { key: string; name: string }
  return { key: meta.key, name: meta.name }
}

export async function getDownloadUrl(apiClient: ApiClient, key: string): Promise<{ url: string; expiresAt: number }> {
  const response = await fetch(`${API_BASE_URL}/uploads/${key}`, {
    headers: await apiClient.getAuthHeaders(false),
  })

  if (!response.ok) {
    throw new Error(`Failed to get download URL: ${response.status} ${response.statusText}`)
  }

  const data = (await response.json()) as { download_url: string; expires_at: number }
  const url = data.download_url.startsWith('/')
    ? `${new URL(API_BASE_URL).origin}${data.download_url}`
    : data.download_url
  return { url, expiresAt: data.expires_at }
}
