import { useState, useRef, useEffect, type DragEvent, type ChangeEvent } from 'react';
import { Card, Flex, Text, Box, IconButton } from '@radix-ui/themes';
import { UploadIcon, FileTextIcon, Cross2Icon, CheckCircledIcon, ExclamationTriangleIcon } from '@radix-ui/react-icons';
import { uploadFile, type FileMetadata } from '../api';

interface FileWidgetProps {
    label: string;
    name: string;
    onChange: (fileMetadata: FileMetadata | null) => void;
    value?: FileMetadata | null;
    accept?: string;
    maxSizeMB?: number; // Defaults to 5MB
    disabled?: boolean;
    hint?: string;
}

// Helper to generate a friendly string from accept types
const formatAccept = (accept: string) => {
    if (!accept || accept === '*/*') return 'Files';
    if (accept.includes('image/*')) return 'Images';
    return accept.split(',').map(t => {
        const parts = t.trim().split('/');
        // Handle extension-only types (e.g. .pdf) if passed, though accept usually expects mime types
        if (parts[0].startsWith('.')) return parts[0].substring(1).toUpperCase();
        return parts[1] ? parts[1].toUpperCase() : parts[0];
    }).join(', ');
};

const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 Bytes';
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
};

export function FileWidget({
    label,
    name,
    onChange,
    value,
    accept = 'image/*,application/pdf',
    maxSizeMB = 5,
    disabled = false,
    hint,
}: FileWidgetProps) {
    const [dragActive, setDragActive] = useState(false);
    const [progress, setProgress] = useState(0);
    const [status, setStatus] = useState<'IDLE' | 'UPLOADING' | 'SUCCESS' | 'ERROR'>('IDLE');
    const [errorMessage, setErrorMessage] = useState<string | null>(null);
    const inputRef = useRef<HTMLInputElement>(null);
    const abortControllerRef = useRef<AbortController | null>(null);

    // Abort on unmount
    useEffect(() => {
        return () => {
            abortControllerRef.current?.abort();
        };
    }, []);

    const handleDrag = (e: DragEvent<HTMLDivElement>) => {
        e.preventDefault();
        e.stopPropagation();
        if (disabled || value) return;

        if (e.type === 'dragenter' || e.type === 'dragover') {
            setDragActive(true);
        } else if (e.type === 'dragleave') {
            setDragActive(false);
        }
    };

    const validateFile = (file: File): string | null => {
        // Validate Size
        if (file.size > maxSizeMB * 1024 * 1024) {
            return `File size exceeds ${maxSizeMB}MB limit.`;
        }
        // Validate Type (Simple check)
        // Note: 'accept' prop is mainly for file picker, manual validation for drag-drop is good practice
        // but regex matching mime types can be complex. We'll rely on server validation mostly,
        // but can do simple checks here if needed.
        return null;
    };

    const handleUpload = async (file: File) => {
        const error = validateFile(file);
        if (error) {
            setErrorMessage(error);
            setStatus('ERROR');
            return;
        }

        setStatus('UPLOADING');
        setProgress(0);
        setErrorMessage(null);

        // Cancel any previous upload
        abortControllerRef.current?.abort();
        const controller = new AbortController();
        abortControllerRef.current = controller;

        try {
            const metadata = await uploadFile(file, (p) => setProgress(p), controller.signal);
            setStatus('SUCCESS');
            onChange(metadata);
        } catch (err) {
            if (err instanceof Error && err.name === 'AbortError') {
                console.log('Upload aborted.');
                return;
            }
            console.error(err);
            setStatus('ERROR');
            setErrorMessage(err instanceof Error ? err.message : 'Upload failed');
        }
    };

    const handleDrop = (e: DragEvent<HTMLDivElement>) => {
        e.preventDefault();
        e.stopPropagation();
        setDragActive(false);
        if (disabled || value) return;

        if (e.dataTransfer.files && e.dataTransfer.files[0]) {
            handleUpload(e.dataTransfer.files[0]);
        }
    };

    const handleChange = (e: ChangeEvent<HTMLInputElement>) => {
        e.preventDefault();
        if (e.target.files && e.target.files[0]) {
            handleUpload(e.target.files[0]);
        }
    };

    const handleRemove = () => {
        if (status === 'UPLOADING') {
            abortControllerRef.current?.abort();
        }
        onChange(null);
        setStatus('IDLE');
        setProgress(0);
        setErrorMessage(null);
        if (inputRef.current) {
            inputRef.current.value = '';
        }
    };

    const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
        if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            inputRef.current?.click();
        }
    };

    return (
        <Box>
            <Text as="label" size="2" weight="bold" mb="1" className="block">
                {label}
            </Text>

            {value ? (
                // File Present State
                <Card size="2" variant="surface" className="relative group">
                    <Flex align="center" gap="3">
                        <Box className="bg-blue-100 p-2 rounded text-blue-600">
                            <FileTextIcon width="20" height="20" />
                        </Box>
                        <Box style={{ flex: 1, overflow: 'hidden' }}>
                            <Text size="2" weight="bold" className="block truncate">
                                {value.name}
                            </Text>
                            <Text size="1" color="gray">
                                {formatBytes(value.size)}
                            </Text>
                        </Box>
                        <Flex align="center" gap="2">
                            <CheckCircledIcon className="text-green-600 w-5 h-5" />
                            {!disabled && (
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
                    <input type="hidden" name={name} value={value.key} />
                </Card>
            ) : (
                // Upload State
                <div
                    className={`
            border-2 border-dashed rounded-lg p-6 text-center transition-all duration-200 ease-in-out
            ${dragActive ? 'border-blue-500 bg-blue-50' : 'border-gray-300 hover:border-blue-400 hover:bg-gray-50'}
            ${status === 'ERROR' ? 'border-red-300 bg-red-50' : ''}
            ${disabled ? 'opacity-60 cursor-not-allowed pointer-events-none' : 'cursor-pointer'}
          `}
                    onDragEnter={handleDrag}
                    onDragLeave={handleDrag}
                    onDragOver={handleDrag}
                    onDrop={handleDrop}
                    onClick={() => inputRef.current?.click()}
                    onKeyDown={handleKeyDown}
                    role="button"
                    tabIndex={disabled ? -1 : 0}
                >
                    <input
                        ref={inputRef}
                        type="file"
                        className="hidden"
                        accept={accept}
                        onChange={handleChange}
                        disabled={disabled}
                    />

                    <Flex direction="column" align="center" gap="2">
                        {status === 'UPLOADING' ? (
                            <Box className="w-full max-w-[200px]">
                                <Text size="2" weight="bold" color="blue" mb="2">
                                    Uploading... {Math.round(progress)}%
                                </Text>
                                <div className="w-full bg-gray-200 rounded-full h-2.5 dark:bg-gray-700">
                                    <div
                                        className="bg-blue-600 h-2.5 rounded-full transition-all duration-300"
                                        style={{ width: `${progress}%` }}
                                    ></div>
                                </div>
                            </Box>
                        ) : status === 'ERROR' ? (
                            <>
                                <ExclamationTriangleIcon className="w-8 h-8 text-red-500" />
                                <Text size="2" color="red" weight="medium">
                                    {errorMessage || 'Upload failed'}
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
                                    {hint || `${formatAccept(accept)} up to ${maxSizeMB}MB`}
                                </Text>
                            </>
                        )}
                    </Flex>
                </div>
            )}
        </Box>
    );
}
