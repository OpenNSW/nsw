import type { ApiClient } from '../api'
import { getEnv } from '../runtimeConfig'

const API_BASE_URL = getEnv('VITE_API_BASE_URL', 'http://localhost:8080/api/v1')!

export interface UploadResponse {
  key: string
  name: string
  url?: string
}

export async function uploadFile(apiClient: ApiClient, file: File): Promise<UploadResponse> {
  // Request the presigned URL (S3 or local equivalent)
  const initResponse = await fetch(`${API_BASE_URL}/uploads`, {
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

  if (!initResponse.ok) {
    const errorText = await initResponse.text()
    console.error(`Upload initialization error ${initResponse.status}: ${errorText}`)
    throw new Error(`Failed to initialize upload: ${initResponse.status}`)
  }

  const meta = (await initResponse.json()) as { key: string; name: string; upload_url: string; url: string }

  if (!meta.upload_url) {
    throw new Error('Server did not provide an upload URL')
  }

  // Upload directly to Blob Storage / Local Mock
  const uploadResponse = await fetch(meta.upload_url, {
    method: 'PUT',
    headers: {
      'Content-Type': file.type || 'application/octet-stream',
    },
    body: file,
  })

  if (!uploadResponse.ok) {
    const errorText = await uploadResponse.text()
    console.error(`File content upload error ${uploadResponse.status}: ${errorText}`)
    throw new Error(`Failed to upload file content: ${uploadResponse.status}`)
  }

  return { key: meta.key, name: meta.name, url: meta.url }
}

export async function getDownloadUrl(apiClient: ApiClient, key: string): Promise<{ url: string; expiresAt: number }> {
  const response = await fetch(`${API_BASE_URL}/uploads/${key}`, {
    headers: await apiClient.getAuthHeaders(false),
  })

  if (!response.ok) {
    throw new Error(`Failed to get download URL: ${response.status}`)
  }

  const data = (await response.json()) as { download_url: string; expires_at: number }

  // Format local relative paths to absolute paths so the browser can download it
  const url = data.download_url.startsWith('/')
    ? `${new URL(API_BASE_URL).origin}${data.download_url}`
    : data.download_url

  return { url, expiresAt: data.expires_at }
}
