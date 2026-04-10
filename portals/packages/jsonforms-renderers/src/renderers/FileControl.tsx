import { withJsonFormsControlProps } from '@jsonforms/react';
import type { ControlElement, JsonSchema } from '@jsonforms/core';
import { Card, Flex, Text, Box, IconButton, Button } from '@radix-ui/themes';
import {
    UploadIcon, FileTextIcon, Cross2Icon,
    CheckCircledIcon, ExclamationTriangleIcon
} from '@radix-ui/react-icons';
import {
    useState, useRef, useEffect, useCallback,
    type ChangeEvent, type DragEvent
} from 'react';
import { useUpload } from '../contexts/UploadContext';
import * as React from 'react';

interface FileEntry {
    key: string;
    name: string;
    blobUrl?: string;
}

interface XFileOptions {
    maxFiles?: number;
    maxSize?: number;
    accept?: string;
}

interface FileControlProps {
    data: string | string[] | null;
    handleChange(path: string, value: string | string[] | null): void;
    path: string;
    label: string;
    required?: boolean;
    uischema?: ControlElement;
    schema?: JsonSchema & { 'x-file'?: XFileOptions };
    enabled?: boolean;
}

function normalizeData(data: string | string[] | null): string[] {
    if (!data) return [];
    return Array.isArray(data) ? data : [data];
}

function formatBytes(bytes: number): string {
    if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(0)} MB`;
    if (bytes >= 1024) return `${(bytes / 1024).toFixed(0)} KB`;
    return `${bytes} B`;
}

function formatAccept(accept: string): string {
    return accept
        .split(',')
        .map(t => {
            t = t.trim();
            if (t === 'image/*') return 'Images';
            if (t === 'application/pdf') return 'PDF';
            if (t.startsWith('.')) return t.toUpperCase();
            return t.split('/').pop()?.toUpperCase() ?? t;
        })
        .join(', ');
}

const FileControl = ({
                         data, handleChange, path, label, required,
                         uischema, schema, enabled
                     }: FileControlProps) => {
    const uploadContext = useUpload();

    // Read x-file constraints from schema
    const xFile: XFileOptions = schema?.['x-file'] ?? {};
    const uiOptions = uischema?.options ?? {};

    const maxFiles = xFile.maxFiles ?? (uiOptions.maxFiles as number) ?? 1;
    const maxSize  = xFile.maxSize  ?? (uiOptions.maxSize  as number) ?? 5 * 1024 * 1024;
    const accept   = xFile.accept   ?? (uiOptions.accept   as string) ?? 'image/*,application/pdf';

    const isMulti   = maxFiles > 1;
    const isEnabled = enabled !== false;

    const [dragActive, setDragActive]   = useState(false);
    const [error, setError]             = useState<string | null>(null);
    const [fileEntries, setFileEntries] = useState<Record<string, FileEntry>>({});
    const inputRef = useRef<HTMLInputElement>(null);

    const currentKeys = normalizeData(data);
    const atLimit     = currentKeys.length >= maxFiles;

    useEffect(() => {
        return () => {
            Object.values(fileEntries).forEach(e => {
                if (e.blobUrl) URL.revokeObjectURL(e.blobUrl);
            });
        };
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const processFile = useCallback(async (file: File) => {
        setError(null);

        if (currentKeys.length >= maxFiles) {
            setError(`Maximum ${maxFiles} file${maxFiles > 1 ? 's' : ''} allowed.`);
            return;
        }
        if (file.size > maxSize) {
            setError(`File exceeds the ${formatBytes(maxSize)} limit.`);
            return;
        }

        const acceptedTypes = accept.split(',').map(t => t.trim());
        const typeOk = acceptedTypes.some(type => {
            if (type === '*' || type === '*/*') return true;
            if (type.endsWith('/*')) return file.type.startsWith(type.slice(0, -1));
            if (type.startsWith('.')) return file.name.toLowerCase().endsWith(type.toLowerCase());
            return file.type === type;
        });
        if (!typeOk) {
            setError(`Invalid type. Accepted: ${formatAccept(accept)}`);
            return;
        }

        if (!uploadContext?.onUpload) {
            setError('Upload service not configured.');
            return;
        }

        try {
            const result  = await uploadContext.onUpload(file);
            const blobUrl = URL.createObjectURL(file);
            const entry: FileEntry = { key: result.key, name: result.name ?? file.name, blobUrl };

            setFileEntries(prev => ({ ...prev, [result.key]: entry }));

            const newKeys = [...currentKeys, result.key];
            handleChange(path, isMulti ? newKeys : newKeys[0]);
        } catch {
            setError('Upload failed. Please try again.');
            if (inputRef.current) inputRef.current.value = '';
        }
    }, [currentKeys, maxFiles, maxSize, accept, uploadContext, path, handleChange, isMulti]);

    const handleDrag = (e: DragEvent<HTMLDivElement>) => {
        e.preventDefault(); e.stopPropagation();
        if (!isEnabled || atLimit) return;
        setDragActive(e.type === 'dragenter' || e.type === 'dragover');
    };

    const handleDrop = (e: DragEvent<HTMLDivElement>) => {
        e.preventDefault(); e.stopPropagation();
        setDragActive(false);
        if (!isEnabled || atLimit) return;
        if (e.dataTransfer.files?.[0]) processFile(e.dataTransfer.files[0]);
    };

    const handleInputChange = (e: ChangeEvent<HTMLInputElement>) => {
        if (e.target.files?.[0]) {
            processFile(e.target.files[0]);
            e.target.value = '';
        }
    };

    const handleRemove = (key: string) => {
        if (!isEnabled) return;
        setFileEntries(prev => {
            const next = { ...prev };
            if (next[key]?.blobUrl) URL.revokeObjectURL(next[key].blobUrl!);
            delete next[key];
            return next;
        });
        const newKeys = currentKeys.filter(k => k !== key);
        handleChange(path, isMulti
            ? (newKeys.length > 0 ? newKeys : null)
            : (newKeys[0] ?? null)
        );
    };

    const onView = async (e: React.MouseEvent<HTMLButtonElement>, key: string) => {
        e.preventDefault();
        const blobUrl = fileEntries[key]?.blobUrl;
        if (blobUrl) {
            window.open(blobUrl, '_blank', 'noopener,noreferrer')?.focus();
            return;
        }
        const newWindow = window.open('', '_blank');
        if (!newWindow) return;
        try {
            const result = await uploadContext?.getDownloadUrl?.(key);
            if (result?.url) newWindow.location.href = result.url;
            else newWindow.close();
        } catch { newWindow.close(); }
    };

    if (!isEnabled && currentKeys.length === 0) return null;

    const remaining    = maxFiles - currentKeys.length;
    const showDropZone = isEnabled && !atLimit;

    return (
        <Box mb="4">
            {/* ── Header row ── */}
            <Flex align="center" justify="between" mb="2">
                <Text as="label" size="2" weight="bold">
                    {label}{required && <Text color="red"> *</Text>}
                </Text>
                <Text size="1" color="gray">
                    {isMulti
                        ? `Up to ${maxFiles} files · ${formatBytes(maxSize)} each · ${formatAccept(accept)}`
                        : `${formatBytes(maxSize)} · ${formatAccept(accept)}`
                    }
                </Text>
            </Flex>

            {/* ── Uploaded file rows ── */}
            {currentKeys.map(key => (
                <Card key={key} size="2" variant="surface" mb="2">
                    <Flex align="center" gap="3">
                        <Box style={{
                            background: 'var(--blue-3)',
                            padding: 8,
                            borderRadius: 6,
                            color: 'var(--blue-9)',
                            flexShrink: 0
                        }}>
                            <FileTextIcon width="20" height="20" />
                        </Box>
                        <Box style={{ flex: 1, overflow: 'hidden' }}>
                            <Text size="2" weight="bold" style={{
                                display: 'block',
                                overflow: 'hidden',
                                textOverflow: 'ellipsis',
                                whiteSpace: 'nowrap'
                            }}>
                                {fileEntries[key]?.name ?? 'Uploaded File'}
                            </Text>
                        </Box>
                        <Flex align="center" gap="2" style={{ flexShrink: 0 }}>
                            <Button variant="soft" color="blue" size="1"
                                    onClick={e => onView(e, key)}>
                                View
                            </Button>
                            <CheckCircledIcon style={{ color: 'var(--green-9)', width: 18, height: 18 }} />
                            {isEnabled && (
                                <IconButton variant="ghost" color="gray"
                                            onClick={() => handleRemove(key)}>
                                    <Cross2Icon />
                                </IconButton>
                            )}
                        </Flex>
                    </Flex>
                </Card>
            ))}

            {/* ── Drop zone — hidden once limit reached ── */}
            {showDropZone && (
                <div
                    role="button"
                    tabIndex={0}
                    onClick={() => inputRef.current?.click()}
                    onKeyDown={e => {
                        if (e.key === 'Enter' || e.key === ' ') {
                            e.preventDefault();
                            inputRef.current?.click();
                        }
                    }}
                    onDragEnter={handleDrag}
                    onDragLeave={handleDrag}
                    onDragOver={handleDrag}
                    onDrop={handleDrop}
                    style={{ cursor: 'pointer' }}
                    className={[
                        'border-2 border-dashed rounded-lg p-6 text-center',
                        'transition-all duration-200 ease-in-out',
                        dragActive
                            ? 'border-blue-500 bg-blue-50'
                            : error
                                ? 'border-red-300 bg-red-50'
                                : 'border-gray-300 hover:border-blue-400 hover:bg-gray-50'
                    ].join(' ')}
                >
                    <input
                        ref={inputRef}
                        type="file"
                        style={{ display: 'none' }}
                        accept={accept}
                        onChange={handleInputChange}
                    />
                    <Flex direction="column" align="center" gap="2">
                        {error ? (
                            <>
                                <ExclamationTriangleIcon style={{ width: 32, height: 32, color: 'var(--red-9)' }} />
                                <Text size="2" color="red" weight="medium">{error}</Text>
                                <Text size="1" color="gray">Click to try again</Text>
                            </>
                        ) : (
                            <>
                                <UploadIcon style={{ width: 32, height: 32, color: 'var(--gray-8)' }} />
                                <Text size="2" weight="medium">
                                    {currentKeys.length > 0
                                        ? `Add another file (${remaining} remaining)`
                                        : 'Click to upload or drag and drop'
                                    }
                                </Text>
                                <Text size="1" color="gray">
                                    {formatBytes(maxSize)} max · {formatAccept(accept)}
                                </Text>
                            </>
                        )}
                    </Flex>
                </div>
            )}

            {/* ── Limit reached nudge ── */}
            {isEnabled && atLimit && isMulti && (
                <Text size="1" color="gray" mt="1"
                      style={{ display: 'block', textAlign: 'center' }}>
                    Maximum {maxFiles} files uploaded. Remove one to replace it.
                </Text>
            )}
        </Box>
    );
};

export default withJsonFormsControlProps(FileControl);