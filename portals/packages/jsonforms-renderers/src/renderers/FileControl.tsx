import { withJsonFormsControlProps } from '@jsonforms/react';
import type { ControlElement, JsonSchema } from '@jsonforms/core';
import { Card, Flex, Text, Box, IconButton } from '@radix-ui/themes';
import { UploadIcon, FileTextIcon, Cross2Icon, CheckCircledIcon, ExclamationTriangleIcon } from '@radix-ui/react-icons';
import { useState, useRef, useEffect, useCallback, type ChangeEvent, type DragEvent } from 'react';

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

// Simple in-memory cache for resolved download URLs to avoid redundant API calls
const downloadUrlCache = new Map<string, { url: string; expiresAt: number }>();

/**
 * Checks whether the data value is a file key (UUID-like) rather than a data URL
 */
function isFileKey(data: string): boolean {
    return !data.startsWith('data:');
}

const DEFAULT_API_BASE_URL = 'http://localhost:8080/api/v1';
// TODO: Remove after implementing proper authentication
const TRADER_ID = 'TRADER-001';

const FileControl = ({ data, handleChange, path, label, required, uischema, enabled }: FileControlProps) => {
    const [dragActive, setDragActive] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [fileName, setFileName] = useState<string | null>(null);
    const [downloadUrl, setDownloadUrl] = useState<string | null>(null);
    const [downloadLoading, setDownloadLoading] = useState(false);
    const [downloadError, setDownloadError] = useState<string | null>(null);
    const inputRef = useRef<HTMLInputElement>(null);

    // Get options from UI schema (or default)
    const options = uischema?.options || {};
    const maxSize = (options.maxSize as number) || 5 * 1024 * 1024; // Default 5MB
    const accept = (options.accept as string) || 'image/*,application/pdf';
    const apiBaseUrl = (options.apiBaseUrl as string) || DEFAULT_API_BASE_URL;
    const isEnabled = enabled !== false;

    // Fetch the presigned download URL when data is a file key (not a data URL)
    const fetchDownloadUrl = useCallback(async (fileKey: string) => {
        // Check cache first — use cached URL if not expired (with 60s buffer)
        const cached = downloadUrlCache.get(fileKey);
        if (cached && cached.expiresAt > Date.now() / 1000 + 60) {
            setDownloadUrl(cached.url);
            return;
        }

        setDownloadLoading(true);
        setDownloadError(null);

        try {
            const response = await fetch(`${apiBaseUrl}/uploads/${fileKey}`, {
                headers: { 'Authorization': TRADER_ID },
            });

            if (response.status === 401) {
                setDownloadError('Unauthorized — please log in to download this file.');
                return;
            }

            if (!response.ok) {
                setDownloadError('Failed to generate download link.');
                return;
            }

            const result = await response.json() as { download_url: string; expires_at: number };
            downloadUrlCache.set(fileKey, { url: result.download_url, expiresAt: result.expires_at });
            setDownloadUrl(result.download_url);
        } catch {
            setDownloadError('Unable to reach the server.');
        } finally {
            setDownloadLoading(false);
        }
    }, [apiBaseUrl]);

    useEffect(() => {
        if (data && isFileKey(data) && !isEnabled) {
            fetchDownloadUrl(data);
        } else {
            setDownloadUrl(null);
            setDownloadError(null);
        }
    }, [data, isEnabled, fetchDownloadUrl]);

    const getDisplayText = () => {
        if (fileName) return fileName;
        if (!data) return null;
        // Try to extract name from data URL if stored there, otherwise generic
        return 'Uploaded File';
    };

    // Resolve the href for the download link: use fetched presigned URL for file keys, or data URL directly
    const resolvedHref = data && isFileKey(data) ? downloadUrl : data;

    const processFile = (file: File) => {
        if (file.size > maxSize) {
            const sizeMB = (maxSize / (1024 * 1024)).toFixed(0);
            setError(`File size exceeds ${sizeMB}MB limit.`);
            return;
        }

        const acceptedTypes = accept.split(',').map((t: string) => t.trim());
        const isFileTypeAccepted = acceptedTypes.some((type: string) => {
            if (type.endsWith('/*')) {
                return file.type.startsWith(type.slice(0, -1));
            }
            if (type.startsWith('.')) {
                return file.name.toLowerCase().endsWith(type.toLowerCase());
            }
            return file.type === type;
        });

        // Basic MIME type check (client-side only)
        if (accept !== '*' && !isFileTypeAccepted && !accept.includes('*/*')) {
            setError(`Invalid file type. Accepted types: ${accept}`);
            return;
        }

        const reader = new FileReader();
        reader.onload = () => {
            const result = reader.result as string;
            handleChange(path, result);
            setFileName(file.name);
            setError(null);
        };
        reader.onerror = () => {
            setError('Failed to read file');
        };
        reader.readAsDataURL(file);
    };

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
                            {!isEnabled ? (
                                downloadError ? (
                                    <Text size="1" color="red">{downloadError}</Text>
                                ) : downloadLoading ? (
                                    <Text size="1" color="gray">Generating download link…</Text>
                                ) : (
                                    <a
                                        href={resolvedHref || '#'}
                                        download={fileName || 'document'}
                                        className="text-xs text-blue-600 hover:underline cursor-pointer"
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            if (!resolvedHref) e.preventDefault();
                                        }}
                                    >
                                        Download
                                    </a>
                                )
                            ) : (
                                <Text size="1" color="gray">
                                    Ready to submit
                                </Text>
                            )}
                        </Box>
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
                        className="hidden"
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