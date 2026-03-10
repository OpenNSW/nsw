import { withJsonFormsControlProps } from '@jsonforms/react';
import type { ControlElement, JsonSchema } from '@jsonforms/core';
import { Card, Flex, Text, Box, IconButton, Button } from '@radix-ui/themes';
import { UploadIcon, FileTextIcon, Cross2Icon, CheckCircledIcon, ExclamationTriangleIcon } from '@radix-ui/react-icons';
import { useState, useRef, useEffect, useCallback, type ChangeEvent, type DragEvent } from 'react';
import { useUpload } from '../contexts/UploadContext';

interface FileControlProps {
    data: string | null;
    handleChange(path: string, value: string | null): void;
    path: string;
    label: string;
    required?: boolean;
    uischema?: ControlElement;
    schema?: JsonSchema;
    enabled?: boolean;
}

const MAX_CACHE_SIZE = 50;
const CACHE_TTL_BUFFER_SEC = 60;

/** Bounded cache for download URLs: evict expired on read, cap size when writing. */
const downloadUrlCache = new Map<string, { url: string; expiresAt: number }>();

function evictExpiredCache(): void {
    const now = Date.now() / 1000;
    for (const [k, v] of downloadUrlCache.entries()) {
        if (v.expiresAt <= now + CACHE_TTL_BUFFER_SEC) downloadUrlCache.delete(k);
    }
}

function setCachedDownloadUrl(key: string, url: string, expiresAt: number): void {
    evictExpiredCache();
    if (downloadUrlCache.size >= MAX_CACHE_SIZE) {
        const first = downloadUrlCache.keys().next().value;
        if (first != null) downloadUrlCache.delete(first);
    }
    downloadUrlCache.set(key, { url, expiresAt });
}

function isFileKey(data: string): boolean {
    return !data.startsWith('data:');
}

const FileControl = ({ data, handleChange, path, label, required, uischema, enabled }: FileControlProps) => {
    const uploadContext = useUpload();
    const [dragActive, setDragActive] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [fileName, setFileName] = useState<string | null>(null);
    const [downloadUrl, setDownloadUrl] = useState<string | null>(null);
    const [downloadLoading, setDownloadLoading] = useState(false);
    const [downloadError, setDownloadError] = useState<string | null>(null);
    const [localBlobUrl, setLocalBlobUrl] = useState<string | null>(null);
    const inputRef = useRef<HTMLInputElement>(null);

    useEffect(() => {
        return () => {
            if (localBlobUrl) URL.revokeObjectURL(localBlobUrl);
        };
    }, [localBlobUrl]);

    const options = uischema?.options || {};
    const maxSize = (options.maxSize as number) || 5 * 1024 * 1024; // Default 5MB
    const accept = (options.accept as string) || 'image/*,application/pdf';
    const isEnabled = enabled !== false;

    // Use only the host's getDownloadUrl callback — no fetch in the renderer.
    const fetchDownloadUrl = useCallback(async (fileKey: string, signal?: AbortSignal) => {
        evictExpiredCache();
        const cached = downloadUrlCache.get(fileKey);
        if (cached && cached.expiresAt > Date.now() / 1000 + CACHE_TTL_BUFFER_SEC) {
            setDownloadUrl(cached.url);
            return;
        }

        if (!uploadContext?.getDownloadUrl) {
            if (import.meta.env.DEV) {
                console.warn('[FileControl] UploadProvider should provide getDownloadUrl so the host app can resolve download URLs.');
            }
            setDownloadError('Download not configured for this application.');
            return;
        }

        setDownloadLoading(true);
        setDownloadError(null);

        try {
            const result = await uploadContext.getDownloadUrl(fileKey);
            if (signal?.aborted) return;
            setDownloadUrl(result.url);
            setCachedDownloadUrl(fileKey, result.url, result.expiresAt);
        } catch (e) {
            if (signal?.aborted) return;
            setDownloadError('Unable to reach the server.');
        } finally {
            if (!signal?.aborted) setDownloadLoading(false);
        }
    }, [uploadContext]);

    useEffect(() => {
        if (data && isFileKey(data) && !localBlobUrl) {
            const ac = new AbortController();
            fetchDownloadUrl(data, ac.signal);
            return () => ac.abort();
        }
        if (!localBlobUrl) {
            setDownloadUrl(null);
            setDownloadError(null);
        }
    }, [data, fetchDownloadUrl, localBlobUrl]);

    const getDisplayText = () => {
        if (fileName) return fileName;
        if (!data) return null;
        // Try to extract name from data URL if stored there, otherwise generic
        return 'Uploaded File';
    };

    const [blobUrl, setBlobUrl] = useState<string | null>(null);

    // Convert data URLs to blob URLs to bypass browser restrictions on data: URLs in new tabs
    useEffect(() => {
        if (data && !isFileKey(data)) {
            try {
                const parts = data.split(',');
                const mime = parts[0].match(/:(.*?);/)?.[1] || 'application/octet-stream';
                const b64Data = parts[1];
                const byteCharacters = atob(b64Data);
                const byteNumbers = new Array(byteCharacters.length);
                for (let i = 0; i < byteCharacters.length; i++) {
                    byteNumbers[i] = byteCharacters.charCodeAt(i);
                }
                const byteArray = new Uint8Array(byteNumbers);
                const blob = new Blob([byteArray], { type: mime });
                const url = URL.createObjectURL(blob);
                setBlobUrl(url);

                return () => URL.revokeObjectURL(url);
            } catch (err) {
                console.error('Failed to create blob URL:', err);
                setBlobUrl(null);
            }
        } else {
            setBlobUrl(null);
        }
    }, [data]);

    const resolvedHref = localBlobUrl ?? (data && isFileKey(data) ? downloadUrl : blobUrl);

    const processFile = useCallback(async (file: File) => {
        if (file.size > maxSize) {
            const sizeMB = (maxSize / (1024 * 1024)).toFixed(0);
            setError(`File size exceeds ${sizeMB}MB limit.`);
            return;
        }

        const acceptedTypes = accept.split(',').map((t: string) => t.trim());
        const isFileTypeAccepted = acceptedTypes.some((type: string) => {
            if (type.endsWith('/*')) return file.type.startsWith(type.slice(0, -1));
            if (type.startsWith('.')) return file.name.toLowerCase().endsWith(type.toLowerCase());
            return file.type === type;
        });
        if (accept !== '*' && !isFileTypeAccepted && !accept.includes('*/*')) {
            setError(`Invalid file type. Accepted types: ${accept}`);
            return;
        }

        if (uploadContext?.onUpload) {
            try {
                const result = await uploadContext.onUpload(file);
                setLocalBlobUrl(URL.createObjectURL(file));
                handleChange(path, result.key);
                setFileName(result.name ?? file.name);
                setError(null);
            } catch {
                setError('Upload failed.');
                if (inputRef.current) inputRef.current.value = '';
            }
            return;
        }
        if (import.meta.env.DEV) {
            console.warn('[FileControl] UploadProvider did not supply onUpload; upload service not configured for this application.');
        }
        setError('Upload service not configured for this application.');
    }, [accept, uploadContext, maxSize, path, handleChange]);

    const handleDrag = (e: DragEvent<HTMLDivElement>) => {
        e.preventDefault();
        e.stopPropagation();
        if (!isEnabled || data) return;

        if (e.type === 'dragenter' || e.type === 'dragover') {
            setDragActive(true);
        } else if (e.type === 'dragleave') {
            setDragActive(false);
        }
    };

    const handleDrop = (e: DragEvent<HTMLDivElement>) => {
        e.preventDefault();
        e.stopPropagation();
        setDragActive(false);
        if (!isEnabled || data) return;

        if (e.dataTransfer.files && e.dataTransfer.files[0]) {
            processFile(e.dataTransfer.files[0]);
        }
    };

    const handleInputChange = (e: ChangeEvent<HTMLInputElement>) => {
        if (e.target.files && e.target.files[0]) {
            processFile(e.target.files[0]);
        }
    };

    const handleRemove = () => {
        if (!isEnabled) return;

        if (localBlobUrl) {
            URL.revokeObjectURL(localBlobUrl);
            setLocalBlobUrl(null);
        }
        handleChange(path, null);
        setFileName(null);
        setError(null);
        if (inputRef.current) {
            inputRef.current.value = '';
        }
    };

    const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
        if (isEnabled && (e.key === 'Enter' || e.key === ' ')) {
            e.preventDefault();
            inputRef.current?.click();
        }
    };

    return (
        <Box mb="4">
            <Text as="label" size="2" weight="bold" mb="1" className="block">
                {label} {required && '*'}
            </Text>

            {data ? (
                <Card size="2" variant="surface" className="relative group">
                    <Flex align="center" gap="3">
                        <Box className="bg-blue-100 p-2 rounded text-blue-600">
                            <FileTextIcon width="20" height="20" />
                        </Box>
                        <Box style={{ flex: 1, overflow: 'hidden' }}>
                            <Text size="2" weight="bold" className="block truncate">
                                {getDisplayText()}
                            </Text>
                        </Box>
                        <Flex align="center" gap="3">
                            {downloadLoading ? (
                                <Text size="1" color="gray">Loading...</Text>
                            ) : downloadError ? (
                                <Text size="1" color="red">Error</Text>
                            ) : (
                                <Button variant="soft" color="blue" size="1" asChild>
                                    <a
                                        href={resolvedHref || '#'}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            if (!resolvedHref) e.preventDefault();
                                        }}
                                    >
                                        View
                                    </a>
                                </Button>
                            )}
                            <Flex align="center" gap="2">
                                <CheckCircledIcon className="text-green-600 w-5 h-5" />
                                {isEnabled && (
                                    <IconButton
                                        variant="ghost"
                                        color="gray"
                                        onClick={handleRemove}
                                        className="hover:text-red-600 transition-colors"
                                    >
                                        <Cross2Icon />
                                    </IconButton>
                                )}
                            </Flex>
                        </Flex>
                    </Flex>
                </Card>
            ) : (
                <div
                    className={`
            border-2 border-dashed rounded-lg p-6 text-center transition-all duration-200 ease-in-out
            ${dragActive ? 'border-blue-500 bg-blue-50' : 'border-gray-300 hover:border-blue-400 hover:bg-gray-50'}
            ${error ? 'border-red-300 bg-red-50' : ''}
            ${!isEnabled ? 'opacity-60 cursor-not-allowed pointer-events-none' : 'cursor-pointer'}
          `}
                    onDragEnter={handleDrag}
                    onDragLeave={handleDrag}
                    onDragOver={handleDrag}
                    onDrop={handleDrop}
                    onClick={() => isEnabled && inputRef.current?.click()} // Safety check
                    onKeyDown={handleKeyDown}
                    role="button"
                    tabIndex={!isEnabled ? -1 : 0}
                >
                    <input
                        ref={inputRef}
                        type="file"
                        style={{ display: 'none' }}
                        accept={accept}
                        onChange={handleInputChange}
                        disabled={!isEnabled}
                    />

                    <Flex direction="column" align="center" gap="2">
                        {error ? (
                            <>
                                <ExclamationTriangleIcon className="w-8 h-8 text-red-500" />
                                <Text size="2" color="red" weight="medium">
                                    {error}
                                </Text>
                                <Text size="1" color="gray">Click to try again</Text>
                            </>
                        ) : (
                            <>
                                <UploadIcon className="w-8 h-8 text-gray-400" />
                                <Text size="2" weight="medium">
                                    Click to upload or drag and drop
                                </Text>
                                <Text size="1" color="gray">
                                    Max {Math.round(maxSize / (1024 * 1024))}MB
                                </Text>
                            </>
                        )}
                    </Flex>
                </div>
            )}
        </Box>
    );
};

const FileControlWithProps = withJsonFormsControlProps(FileControl);
export default FileControlWithProps;