const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1'

// TODO: Remove after implementing proper authentication
const TRADER_ID = 'TRADER-001'

export interface FileMetadata {
  id: string
  name: string
  key: string
  url: string
  size: number
  mimeType: string
}

export interface DownloadResponse {
  download_url: string
  expires_at: number
}

export async function uploadFile(file: File): Promise<FileMetadata> {
  const formData = new FormData()
  formData.append('file', file)

  const response = await fetch(`${API_BASE_URL}/uploads`, {
    method: 'POST',
    headers: {
      'Authorization': TRADER_ID,
    },
    body: formData,
  })

  if (!response.ok) {
    const errorText = await response.text()
    console.error(`Upload error ${response.status}: ${errorText}`)
    throw new Error(`Failed to upload file: ${response.status} ${response.statusText}`)
  }

  const metadata = await response.json() as FileMetadata
  return metadata
}

export async function getDownloadUrl(key: string): Promise<string> {
  const response = await fetch(`${API_BASE_URL}/uploads/${key}`, {
    headers: {
      'Authorization': TRADER_ID,
    },
  })

  if (!response.ok) {
    throw new Error(`Failed to get download URL: ${response.status} ${response.statusText}`)
  }

  const data = await response.json() as DownloadResponse
  return data.download_url
}
