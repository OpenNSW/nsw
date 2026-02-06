export interface FileMetadata {
    id: string;
    name: string;
    type: string;
    size: number;
    url: string;
    uploadedAt: string;
}

const API_BASE_URL = (import.meta.env.VITE_API_URL as string | undefined) ?? 'http://localhost:8080';

export async function uploadFile(
    file: File,
    onProgress?: (progress: number) => void,
    signal?: AbortSignal
): Promise<FileMetadata> {
    const formData = new FormData();
    formData.append('file', file);

    // Use XMLHttpRequest to track upload progress if needed, but fetch is simpler.
    // Standard fetch doesn't support upload progress easily.
    // For simplicity, we'll just use fetch and not report granular progress (0 -> 100).

    if (onProgress) onProgress(10); // Start

    const response = await fetch(`${API_BASE_URL}/api/v1/uploads`, {
        method: 'POST',
        body: formData,
        signal,
    });

    if (!response.ok) {
        if (onProgress) onProgress(0);
        throw new Error(`Failed to upload file: ${response.statusText}`);
    }

    const data = (await response.json()) as FileMetadata;
    if (onProgress) onProgress(100); // Finish
    return data as FileMetadata;
}
