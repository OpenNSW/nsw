export const storageKeyRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}(\.[a-z0-9]+)?$/i;

export const getStorageKeyDisplayText = (key: string): string => {
    if (!key) return '';
    if (key.startsWith('data:')) return 'Uploaded File';

    if (storageKeyRegex.test(key)) {
        const parts = key.split('.');
        if (parts.length > 1) {
            return `${parts[parts.length - 1].toUpperCase()} Document`;
        }
        return 'Document';
    }

    return 'Uploaded File';
};

export interface ViewFileOptions {
    localBlobUrl?: string | null;
}

/**
 * Common logic for opening a file/document in a new tab.
 * Handles local blobs, data URLs, and remote storage keys (requires getDownloadUrl).
 */
export const viewFile = async (
    data: string,
    getDownloadUrl?: (key: string) => Promise<{ url: string; expiresAt: number } | null>,
    options: ViewFileOptions = {}
) => {
    // 1. Handle local files immediately (synchronous = no popup block)
    if (options.localBlobUrl) {
        window.open(options.localBlobUrl, '_blank', 'noopener,noreferrer')?.focus();
        return;
    }

    // 2. Handle Data URLs
    if (data && data.startsWith('data:')) {
        let blobUrl: string | null = null;
        try {
            const parts = data.split(',');
            if (parts.length < 2) throw new Error('Invalid data URL');
            
            const mime = parts[0].match(/:(.*?);/)?.[1] || 'application/octet-stream';
            const b64Data = parts[1];
            const byteCharacters = atob(b64Data);
            const byteNumbers = new Array(byteCharacters.length);
            for (let i = 0; i < byteCharacters.length; i++) {
                byteNumbers[i] = byteCharacters.charCodeAt(i);
            }
            const byteArray = new Uint8Array(byteNumbers);
            const blob = new Blob([byteArray], { type: mime });
            blobUrl = URL.createObjectURL(blob);
            window.open(blobUrl, '_blank', 'noopener,noreferrer')?.focus();
        } catch (err) {
            console.error('[StorageUtils] Failed to create blob URL from data URL, attempting direct open:', err);
            window.open(data, '_blank', 'noopener,noreferrer')?.focus();
        } finally {
            // Note: In real production, we might want to delay revocation or let the browser handle it.
        }
        return;
    }

    if (!data) return;

    // 3. Handle remote keys (S3/LocalFS): open a blank tab FIRST to capture the user gesture
    // Then fetch the presigned URL and redirect the tab.
    const newWindow = window.open('', '_blank');
    if (!newWindow) {
        console.warn('[StorageUtils] Popup blocked. Ensure this is called from a user gesture.');
        return;
    }

    try {
        const result = await getDownloadUrl?.(data);
        if (result?.url) {
            newWindow.location.href = result.url;
        } else {
            newWindow.close();
            console.error('[StorageUtils] No download URL returned for key:', data);
        }
    } catch (err) {
        console.error('[StorageUtils] Failed to fetch download URL:', err);
        newWindow.close();
    }
};
